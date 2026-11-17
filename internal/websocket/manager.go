package websocket

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"zjsj/internal/model"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gorilla/websocket"
)

type Scene struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GameConfig struct {
	Name          string `json:"name"`
	MapBase       string `json:"mapBase"`
	LobbyCapacity int    `json:"lobbyCapacity"`
	MapCapacity   int    `json:"mapCapacity"`
	Ip            string `json:"ip"`
	RecordHost    string `json:"recordHost"`
}

type Manager struct {
	mu                sync.RWMutex
	rooms             map[RoomType][]*Room
	clients           map[uint64]*Client
	gameServerClients map[uint64]*Client
	gameConfig        *GameConfig
	userLocks         sync.Map
}

var mapSeq int64
var lobbySeq int64
var gameServerSeq uint64

// 游戏服务器内部 id 基数：多个云服务器进程可能上报重复 id（Unity 包内置 id），
// 冲突时分配内部唯一 id，避免被当成异地登录互踢
const gameServerIdBase uint64 = 1_000_000_000

var recordHost string = "https://game.17vision.com"

// 游戏服务器 IP 缓存文件路径：运行时由连接动态获取，落盘以便重启后兜底
var ipCachePath = gfile.Join(gfile.Pwd(), "storage/config", "ip_cache.json")

func NewManager(lobbyCap, mapCap int) *Manager {
	var gameConfig *GameConfig
	gameConfigPath := gfile.Join(gfile.Pwd(), "storage/config", "game.json")
	gameConfigJson := gfile.GetContents(gameConfigPath)
	if gameConfigJson == "" {
		g.Log("test").Async().Errorf(context.TODO(), "配置文件不存在: %s", gameConfigPath)
		panic("缺少配置文件")
	} else {
		g.Log("test").Async().Infof(context.TODO(), "配置文件: %s", gameConfigJson)
	}

	gjson.Unmarshal([]byte(gameConfigJson), &gameConfig)

	if gameConfig.RecordHost != "" {
		recordHost = gameConfig.RecordHost
	}

	// ip 不再从配置文件读取，改为运行时由游戏服务器客户端连接动态获取；
	// 启动时先用上次缓存的 ip 兜底（若还没有客户端连过，则为空）
	gameConfig.Ip = loadIpCache()

	m := &Manager{
		rooms:             make(map[RoomType][]*Room),
		clients:           make(map[uint64]*Client),
		gameServerClients: make(map[uint64]*Client),
		userLocks:         sync.Map{},
		gameConfig:        gameConfig,
	}

	// 定期清理孤立游戏服务器（Unity 进程已死但 WS 半开等残留）
	go m.reapOrphanServers()
	return m
}

// 读取 IP 缓存文件，返回上次记录的游戏服务器 IP，不存在或解析异常时返回空串
func loadIpCache() string {
	content := gfile.GetContents(ipCachePath)
	if content == "" {
		return ""
	}

	var data struct {
		Ip string `json:"ip"`
	}
	if err := gjson.Unmarshal([]byte(content), &data); err != nil {
		return ""
	}
	return data.Ip
}

// 将游戏服务器 IP 写入缓存文件
func (manager *Manager) saveIpCache(ctx context.Context, ip string) {
	content := gjson.MustEncodeString(map[string]any{
		"ip":        ip,
		"updatedAt": time.Now().Unix(),
	})

	if err := gfile.PutContents(ipCachePath, content); err != nil {
		g.Log("test").Async().Errorf(ctx, "写入 IP 缓存失败: %v", err)
		return
	}
	g.Log("test").Async().Infof(ctx, "已缓存游戏服务器 IP: %s", ip)
}

// 游戏服务器客户端连接时更新 IP：与当前值不同才更新，并写入缓存
func (manager *Manager) updateGameServerIp(ctx context.Context, ip string) {
	if ip == "" {
		return
	}

	manager.mu.Lock()
	if manager.gameConfig.Ip == ip {
		manager.mu.Unlock()
		return
	}
	old := manager.gameConfig.Ip
	manager.gameConfig.Ip = ip
	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "游戏服务器 IP 变更: %q -> %q", old, ip)
	manager.saveIpCache(ctx, ip)
}

// 读取当前游戏服务器 IP（并发安全）
func (manager *Manager) gameServerIp() string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return manager.gameConfig.Ip
}

// 创建一个 Client
func (manager *Manager) createClient(ctx context.Context, user *User, conn *websocket.Conn) *Client {
	// 游戏服务器不走用户的同 id 踢人逻辑：id 冲突时换内部唯一 id，保证多台云服务器共存
	if user.Type == TypeGameServer {
		manager.mu.Lock()
		for {
			_, inClients := manager.clients[user.Id]
			_, inServers := manager.gameServerClients[user.Id]
			if !inClients && !inServers {
				break
			}
			user.Id = gameServerIdBase + atomic.AddUint64(&gameServerSeq, 1)
		}
		manager.mu.Unlock()
	}

	// 按 userId 串行化，避免并发接入互相覆盖或相互踢到对方
	v, _ := manager.userLocks.LoadOrStore(user.Id, &sync.Mutex{})
	um := v.(*sync.Mutex)
	um.Lock()
	defer um.Unlock()

	// 在全局锁内：查找并原子移除旧 client（但不要在持锁时做网络/阻塞操作）
	manager.mu.Lock()
	oldClient, hadOld := manager.clients[user.Id]
	var oldRoom *Room
	if hadOld && oldClient != nil {
		// 从 room 中移除（找到第一个包含该 user 的 room）
		for _, list := range manager.rooms {
			for _, room := range list {
				if _, exists := room.Clients[user.Id]; exists {
					delete(room.Clients, user.Id)
					oldRoom = room
					break
				}
			}

			if oldRoom != nil {
				break
			}
		}
		// 从 manager.clients 中移除旧客户端
		delete(manager.clients, user.Id)
	}

	// 创建并立即注册新 client（保证 manager.clients 总是指向最新 client）
	client := &Client{User: user, conn: conn, send: make(chan []byte, 256), done: make(chan struct{}), CreatedAt: time.Now().Unix(), lastActive: time.Now().Unix()}
	manager.clients[user.Id] = client
	manager.mu.Unlock()

	// 现在在不持全局锁的情况下通知旧连接并关闭资源
	if hadOld && oldClient != nil {
		g.Log("test").Async().Infof(ctx, "%d 被踢下线通知", oldClient.User.Id)

		// 先发通知
		manager.unicastAsync(ctx, oldClient, WSMessage{
			Type: MsgTypeKicked,
			Data: map[string]any{"reason": "异地登录"},
		})

		// 广播旧用户离开旧房间（如果有）
		if oldRoom != nil {
			manager.broadcastAsync(ctx, oldRoom, WSMessage{
				Type: MsgTypeUserLeft,
				Data: UserEventData{RoomId: oldRoom.Id, UserId: oldClient.User.Id},
			}, nil)
		}

		// 关闭旧连接的资源（确保不会阻塞）
		go func(c *Client) {
			// 优雅关闭：关闭 websocket 连接、关闭发送通道等（根据 client.go 的实现调整）
			// 关闭 send chan、done 等需要按你现有 client 实现安全处理
			c.Close()
		}(oldClient)
	}
	return client
}

// 踢掉连接
func (manager *Manager) kickClient(ctx context.Context, user *User, reason string) {
	manager.mu.Lock()
	client, ok := manager.clients[user.Id]
	manager.mu.Unlock()

	if !ok || client == nil {
		return
	}

	g.Log("test").Async().Infof(ctx, "%d 被踢下线通知", client.User.Id)

	// 先发通知，再踢出去
	manager.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeKicked,
		Data: map[string]any{"reason": reason},
	})

	manager.removeClient(ctx, user.Id)
}

// 踢出房间
func (manager *Manager) KickRoom(ctx context.Context, room *Room, user *User, reason string) {
	manager.mu.Lock()
	client, ok := room.Clients[user.Id]
	manager.mu.Unlock()

	if !ok || client == nil {
		return
	}

	g.Log("test").Async().Infof(ctx, "%d 被踢出房间通知", client.User.Id)

	// 先发通知
	manager.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeKickedRoom,
		Data: map[string]any{"reason": reason},
	})

	manager.mu.Lock()
	delete(room.Clients, user.Id)
	empty := len(room.Clients) == 0
	manager.mu.Unlock()

	// 广播用户离开房间
	manager.broadcastUserLeft(ctx, room, user)

	if empty {
		manager.destroyRoom(room)
	}

	// 进入或创建房间
	room = manager.createOrJoinLobby(client)

	// 把房间里的人推送给自己
	manager.pushRoomUserList(ctx, room, *user)

	// 广播消息，有人进来了
	manager.broadcastUserJoined(ctx, room, user)
}

// 游戏服务器加入
func (manager *Manager) addGameClient(client *Client) {
	// 按 userId 串行化，避免并发接入互相覆盖或相互踢到对方
	v, _ := manager.userLocks.LoadOrStore(client.User.Id, &sync.Mutex{})
	um := v.(*sync.Mutex)
	um.Lock()
	defer um.Unlock()

	manager.mu.Lock()
	oldClient, hadOld := manager.gameServerClients[client.User.Id]
	if hadOld && oldClient != nil {
		oldClient.conn.Close()
	}
	manager.gameServerClients[client.User.Id] = client
	manager.mu.Unlock()
}

// 创建或加入一个大厅
func (manager *Manager) createOrJoinLobby(client *Client) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client.RoomType = RoomTypeLobby

	for _, room := range manager.rooms[RoomTypeLobby] {
		if len(room.Clients) < manager.gameConfig.LobbyCapacity {
			client.RoomId = room.Id
			room.Clients[client.User.Id] = client
			return room
		}
	}

	// roomId := fmt.Sprintf("lobby-%d", len(manager.rooms[RoomTypeLobby])+1)
	roomId := fmt.Sprintf("lobby-%d", atomic.AddInt64(&lobbySeq, 1))

	room := &Room{Id: roomId, Name: "大厅", Type: RoomTypeLobby, Capacity: manager.gameConfig.LobbyCapacity, Clients: make(map[uint64]*Client), CreatedAt: time.Now().Unix()}
	client.RoomId = room.Id
	room.Clients[client.User.Id] = client
	manager.rooms[RoomTypeLobby] = append(manager.rooms[RoomTypeLobby], room)
	return room
}

// 获取已经创建好的房间
func (manager *Manager) getMapRoom(mapBase string) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	mapCapacity := manager.gameConfig.MapCapacity

	for _, room := range manager.rooms[RoomTypeMap] {
		if room.MapBase == mapBase && room.Status == RoomStatusWating && len(room.Clients) < mapCapacity {
			return room
		}
	}
	return nil
}

// 创建或进入地图【大厅只有一个，地图却有很多，需要通过 mapBase 来区分】
// 房间满员（最多 MapCapacity 人）就新建房间；新房间触发异步创建云服务器
func (manager *Manager) createOrJoinMap(ctx context.Context, client *Client, mapBase string) *Room {
	manager.mu.Lock()

	client.RoomType = RoomTypeMap

	mapCapacity := manager.gameConfig.MapCapacity

	for _, room := range manager.rooms[RoomTypeMap] {
		// 房间是 wating 状态,并且人数小于设定人数,才可以进(假如存在多个没满,当前逻辑不存在.就应该可以指定房间进的概念)
		if room.MapBase == mapBase && room.Status == RoomStatusWating && len(room.Clients) < mapCapacity {
			client.RoomId = room.Id
			room.Clients[client.User.Id] = client
			// 创建接口曾直接报错的房间（failed 且没拿到 uuid），进房时重试创建
			needRetry := room.ServerStatus == AllocStatusFailed && room.AllocationUuid == ""
			manager.mu.Unlock()
			if needRetry {
				manager.ensureRoomAllocation(ctx, room)
			}
			return room
		}
	}

	rid := nextMapRoomId(mapBase)
	room := &Room{Id: rid, MapBase: mapBase, Type: RoomTypeMap, Name: "地图", Capacity: mapCapacity, Clients: make(map[uint64]*Client), Status: RoomStatusWating, CreatedAt: time.Now().Unix()}

	client.RoomId = room.Id
	room.Clients[client.User.Id] = client
	manager.rooms[RoomTypeMap] = append(manager.rooms[RoomTypeMap], room)
	manager.mu.Unlock()

	// 一个房间一个云服务器：新房间触发异步创建（ip 为空表示创建中）
	manager.ensureRoomAllocation(ctx, room)
	return room
}

// 生成新的地图房间 id
func nextMapRoomId(mapBase string) string {
	return fmt.Sprintf("%s-%d", mapBase, atomic.AddInt64(&mapSeq, 1))
}

// 离开大厅或房间
func (manager *Manager) leftLobbyOrMap(ctx context.Context, client *Client) {
	manager.mu.Lock()

	var room *Room
	for _, r := range manager.rooms[client.RoomType] {
		if r.Id == client.RoomId {
			delete(r.Clients, client.User.Id)
			room = r
			break
		}
	}

	manager.mu.Unlock()

	if room != nil {
		manager.broadcastUserLeft(ctx, room, client.User)

		if len(room.Clients) == 0 {
			manager.destroyRoom(room)
		}
	} else {
		g.Log("test").Async().Infof(ctx, "用户 %d 离开房间失败，找不到房间 %s", client.User.Id, client.RoomId)
	}
}

// 进入某个房间
func (manager *Manager) JoinRoom(ctx context.Context, userId uint64, roomID string) error {
	manager.mu.Lock()
	client, ok := manager.clients[userId]
	if !ok {
		manager.mu.Unlock()
		return fmt.Errorf("user not found")
	}

	var newRoom *Room
	for _, list := range manager.rooms {
		for _, r := range list {
			if r.Id == roomID {
				newRoom = r
				break
			}
		}

		if newRoom != nil {
			break
		}
	}

	if newRoom == nil {
		manager.mu.Unlock()
		return gerror.New("房间不存在")
	}

	if len(newRoom.Clients) >= newRoom.Capacity {
		manager.mu.Unlock()
		return gerror.New("房间已满员")
	}
	manager.mu.Unlock()

	// 先离开房间
	manager.leftLobbyOrMap(ctx, client)

	// 再加入房间
	manager.mu.Lock()
	newRoom.Clients[client.User.Id] = client
	client.RoomId = newRoom.Id
	client.RoomType = newRoom.Type
	manager.mu.Unlock()

	// 把房间里的人推送给自己
	manager.pushRoomUserList(ctx, newRoom, *client.User)

	// 广播消息，有人进来了
	manager.broadcastUserJoined(ctx, newRoom, client.User)

	// 推送房间云服务器状态
	manager.pushServerStatus(ctx, newRoom, client)

	return nil
}

func (manager *Manager) start(ctx context.Context, roomId string) bool {
	// 先在全局锁内查找并修改房间状态（写操作需锁）
	manager.mu.Lock()

	var room *Room
	for _, list := range manager.rooms {
		for _, r := range list {
			if r.Id == roomId {
				room = r
				break
			}
		}

		if room != nil {
			break
		}
	}

	if room == nil {
		manager.mu.Unlock()
		g.Log("test").Async().Infof(ctx, "开始游戏失败，房间不存在: %s", roomId)
		return false
	}

	// 房间必须有人才能开始游戏
	if len(room.Clients) == 0 {
		manager.mu.Unlock()
		g.Log("test").Async().Infof(ctx, "开始游戏失败，房间无人: %s", roomId)
		return false
	}

	// 云服务器必须就绪（有 ip 且游戏服务器已回连）才能开始
	if room.Type == RoomTypeMap && (room.Ip == "" || room.GameServerClient == nil) {
		manager.mu.Unlock()
		g.Log("test").Async().Infof(ctx, "开始游戏失败，云服务器未就绪, 房间 %s", roomId)
		return false
	}

	// 修改房间状态并记录时间（在锁内完成）
	room.Status = RoomStatusPlaying
	room.StartTime = time.Now().Unix()

	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "开始游戏, 房间 id = %s, 新状态 = %v", roomId, RoomStatusPlaying)

	// 带 roomId/startTime 载荷：Data 为 nil 时 omitempty 会丢掉 data 字段，Unity 解析空 data 易崩
	startData := map[string]any{"roomId": room.Id, "startTime": room.StartTime}

	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeStartGame,
		Data: startData,
	}, nil)

	// 通知游戏服务器开始游戏
	if room.GameServerClient != nil {
		manager.unicastAsync(ctx, room.GameServerClient, WSMessage{
			Type: MsgTypeStartGame,
			Data: startData,
		})

		g.Log("test").Async().Infof(ctx, "开始游戏 给游戏客户端发消息。用户 id = %d, 端口 = %d, ", room.GameServerClient.User.Id, room.GameServerClient.User.Port)
	} else {
		g.Log("test").Async().Error(ctx, "开始游戏失败, 游戏客户端不存在")
	}

	// 设置房间用户状态，提交服务端统计
	for _, client := range room.Clients {
		client.User.Extend.StartTime = time.Now().Unix()

		data := &model.GameRecordsCreateReq{
			GlassesUseId: client.User.GlassesUseId,
			Name:         defaultMapBase,
			Ip:           client.User.Extend.ConnectIp,
			ConnectAt:    gtime.NewFromTimeStamp(client.User.Extend.ConnectTime),
			StartAt:      gtime.NewFromTimeStamp(client.User.Extend.StartTime),
		}

		res, err := service.GameRecords().Create(ctx, *data)
		if err != nil {
			g.Log("test").Async().Infof(ctx, "[1]写入统计失败:%s", err.Error())
			continue
		}

		if res == nil {
			g.Log("test").Async().Infof(ctx, "[2]写入统计失败:数据为空")
			continue
		}

		client.User.Extend.RecordId = res.Id

		g.Log("test").Async().Infof(ctx, "用户 %d 开始游戏已统计。统计 id 是 %d", client.User.Id, res.Id)
	}

	return true
}

func (manager *Manager) stop(ctx context.Context, roomId string) bool {
	// 先在全局锁内查找并修改房间状态（写操作需锁）
	manager.mu.Lock()

	var room *Room
	for _, list := range manager.rooms {
		for _, r := range list {
			if r.Id == roomId {
				room = r
				break
			}
		}

		if room != nil {
			break
		}
	}

	if room == nil {
		manager.mu.Unlock()
		g.Log("test").Async().Infof(ctx, "结束游戏失败，房间不存在: %s", roomId)
		return false
	}

	// 修改房间状态并记录时间（在锁内完成）
	room.Status = RoomStatusStopped
	room.StartTime = 0

	// 关键：拷贝 clients 引用，避免解锁后遍历原 map
	clients := make([]*Client, 0, len(room.Clients))
	for _, c := range room.Clients {
		clients = append(clients, c)
	}

	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "结束游戏, 房间 id = %s, 新状态 = %v", roomId, RoomStatusStopped)

	stopData := map[string]any{"roomId": room.Id}

	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeStopGame,
		Data: stopData,
	}, nil)

	if room.GameServerClient != nil {
		manager.unicastAsync(ctx, room.GameServerClient, WSMessage{
			Type: MsgTypeStopGame,
			Data: stopData,
		})
	}

	// 异步断开用户连接，不阻塞当前调用
	go func(clients []*Client) {
		time.Sleep(200 * time.Millisecond)
		for _, c := range clients {
			c.conn.Close()
		}
		g.Log("test").Async().Infof(ctx, "已关闭房间 %s 的所有连接", roomId)
	}(clients)

	return true
}

// 告诉客户端，可以进房间：房间满员会新建房间并触发创建云服务器，所以总是可以进
func (manager *Manager) pushUserCanEnterRoom(ctx context.Context, client *Client) {
	manager.unicastAsync(ctx, client, WSMessage{
		Type: CanEnterRoom,
		Data: nil,
	})
}

// 通知大厅用户可以进房间了（房间满员会新建房间）
func (manager *Manager) checkUserCanEnterRoom(ctx context.Context) {
	manager.mu.RLock()
	clients := make([]*Client, 0)
	for _, room := range manager.rooms[RoomTypeLobby] {
		for _, c := range room.Clients {
			clients = append(clients, c)
		}
	}
	manager.mu.RUnlock()

	g.Log("test").Async().Infof(ctx, "通知 %d 个大厅用户可以进入房间", len(clients))

	for _, c := range clients {
		manager.unicastAsync(ctx, c, WSMessage{
			Type: CanEnterRoom,
			Data: nil,
		})
	}
}

// 推送房间用户
func (manager *Manager) pushRoomUserList(ctx context.Context, room *Room, user User) {
	users := make([]*User, 0, len(room.Clients))

	for _, c := range room.Clients {
		if c.User != nil {
			users = append(users, c.User)
		}
	}

	if c, ok := room.Clients[user.Id]; ok {
		manager.unicastAsync(ctx, c, WSMessage{
			Type: MsgTypeRoomUsers,
			Data: RoomUsersData{RoomId: room.Id, User: user, Users: users},
		})
	}
}

// 广播新用户加入
func (manager *Manager) broadcastUserJoined(ctx context.Context, room *Room, user *User) {
	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeUserJoined,
		Data: UserEventData{RoomId: room.Id, UserId: user.Id, Ip: room.Ip},
	}, user)
}

// 广播用户离开房间
func (manager *Manager) broadcastUserLeft(ctx context.Context, room *Room, user *User) {
	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeUserLeft,
		Data: UserEventData{RoomId: room.Id, UserId: user.Id},
	}, user)
}

// 移除 client
func (manager *Manager) removeClient(ctx context.Context, userId uint64) {
	var offlineRoom *Room
	manager.mu.Lock()
	client, ok := manager.clients[userId]
	if !ok {
		// 兜底：clients 里已被先删但 gameServerClients 还残留（竞态遗留），同样要清理
		client, ok = manager.gameServerClients[userId]
	}
	if !ok {
		manager.mu.Unlock()
		return
	}

	g.Log("test").Async().Infof(ctx, "用户 %d 退出", client.User.Id)

	// 只删属于自己的条目，避免误删同 id 的新连接
	if manager.clients[userId] == client {
		delete(manager.clients, userId)
	}

	recordId := client.User.Extend.RecordId

	glassesUseId := client.User.GlassesUseId

	// 如果是游戏服务器挂了，就删除游戏服务器，并解绑房间
	if client.User.Type == TypeGameServer {
		client.User.Extend.IsServer = false
		client.RoomId = ""
		// 删除引用
		if manager.gameServerClients[userId] == client {
			delete(manager.gameServerClients, userId)
		}
		for _, r := range manager.rooms[RoomTypeMap] {
			if r.GameServerClient == client {
				r.GameServerClient = nil
				r.serverNotified = false
				offlineRoom = r
				break
			}
		}
		g.Log("test").Async().Infof(ctx, "游戏服务器退出，删除引用")
	}

	var room *Room
	for _, list := range manager.rooms {
		for _, r := range list {
			if c, exists := r.Clients[userId]; exists && c == client {
				delete(r.Clients, userId)
				room = r
				break
			}
		}

		if room != nil {
			break
		}
	}
	manager.mu.Unlock()

	// 游戏服务器掉线：通知房间用户（进程可能还活着，等待回连重新绑定）
	if offlineRoom != nil {
		manager.broadcastAsync(ctx, offlineRoom, WSMessage{
			Type: MsgTypeServerOffline,
			Data: UserEventData{RoomId: offlineRoom.Id, Message: "游戏服务器掉线，等待回连"},
		}, nil)
	}

	if room != nil {
		manager.broadcastUserLeft(ctx, room, client.User)

		if len(room.Clients) == 0 {
			g.Log("test").Async().Infof(ctx, "房间无人，销毁房间 %s ", room.Id)
			if room.GameServerClient != nil {
				room.GameServerClient.User.Extend.IsServer = false
				room.GameServerClient.RoomId = ""
				room.GameServerClient = nil

				g.Log("test").Async().Info(ctx, "销毁房间，复位数据:")
				g.Log("test").Async().Info(ctx, gjson.MustEncodeString(manager.gameServerClients))
			}

			manager.destroyRoom(room)
		}
	}

	if glassesUseId > 0 {
		apiCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := service.GlassesUse().UpdateStatus(apiCtx, int64(glassesUseId), 3)
		if err != nil {
			g.Log().Error(ctx, "更新设备状态为 3 失败", err.Error())
		} else {
			g.Log().Info(ctx, "更新设备状态为 3 成功")
		}
	}

	if recordId != 0 {
		apiCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		data := &model.GameRecordsUpdateReq{
			Id:    recordId,
			EndAt: gtime.Now().Local(),
		}

		err := service.GameRecords().Update(apiCtx, *data)

		if err != nil {
			g.Log("test").Errorf(apiCtx, "上报结束失败 recordId=%d err=%v", recordId, err)
		} else {
			g.Log("test").Debugf(apiCtx, "上报结束成功 recordId=%d", recordId)
		}
	}
}

// 销毁指定房间，并释放其绑定的云服务器
func (manager *Manager) destroyRoom(room *Room) {
	manager.mu.Lock()
	list := manager.rooms[room.Type]
	filtered := list[:0]
	for _, r := range list {
		if r != room {
			filtered = append(filtered, r)
		}
	}
	manager.rooms[room.Type] = filtered
	allocUuid := room.AllocationUuid
	room.AllocationUuid = ""
	// 解绑游戏服务器：房间没了，连接若还活着则复位为待服务，交给孤儿清理或重新绑定
	if room.GameServerClient != nil {
		room.GameServerClient.User.Extend.IsServer = false
		room.GameServerClient.RoomId = ""
		room.GameServerClient = nil
	}
	manager.mu.Unlock()

	// 房间没了，释放对应的云服务器，避免继续计费
	if allocUuid != "" {
		go func(id string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			unityService.DeleteAllocation(ctx, id)
		}(allocUuid)
	}
}

// 孤立服务器清理参数：每 15s 扫一次，连续孤立 30s 才清理（留回连/补绑定宽限）
const (
	orphanReapInterval = 15 * time.Second
	orphanGrace        = 30 * time.Second
	orphanIdleTimeout  = 60 * time.Second // 空闲服务器心跳超过该时长无响应才判死
)

// reapOrphanServers 定期清理孤立游戏服务器：连接已关闭、绑定房间已销毁/被顶替、或长期无房间可服务。
// Unity 进程已死但 WS 半开（ping 被中间层应答）时读泵不会退出，靠这里兜底 Close 触发清理。
func (manager *Manager) reapOrphanServers() {
	ctx := context.Background()
	ticker := time.NewTicker(orphanReapInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		var victims []*Client

		manager.mu.Lock()
		for _, c := range manager.gameServerClients {
			var bound *Room
			if c.RoomId != "" {
				for _, list := range manager.rooms {
					for _, r := range list {
						if r.Id == c.RoomId {
							bound = r
							break
						}
					}
					if bound != nil {
						break
					}
				}
			}

			// 孤立判定：连接已关 / 绑定的房间不存在或已绑别人 / 无房间可服务
			orphan := c.IsClosed() || (c.RoomId != "" && (bound == nil || bound.GameServerClient != c))
			if !orphan && c.RoomId == "" {
				// 未绑定：心跳正常的健康连接保留待复用；
				// 仅当没有创建中的分配且心跳长期无响应（进程已死/半开）才清理
				orphan = !unityService.HasCreatingAllocation() && now-atomic.LoadInt64(&c.lastActive) > int64(orphanIdleTimeout/time.Second)
			}

			if !orphan {
				c.orphanSince = 0
				continue
			}
			if c.IsClosed() || (c.orphanSince != 0 && now-c.orphanSince >= int64(orphanGrace/time.Second)) {
				victims = append(victims, c)
			} else if c.orphanSince == 0 {
				c.orphanSince = now
			}
		}
		manager.mu.Unlock()

		// Close 触发读泵退出 → removeClient 完成 map 清理与房间解绑
		for _, c := range victims {
			g.Log("test").Async().Infof(ctx, "清理孤立游戏服务器: id=%d, roomId=%s", c.User.Id, c.RoomId)
			c.Close()
		}
	}
}

func (manager *Manager) error(ctx context.Context, client *Client, err error) {
	manager.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeError,
		Data: err.Error(),
	})
}

// 广播消息
func (manager *Manager) broadcastAsync(ctx context.Context, room *Room, msg WSMessage, user *User) {
	dataStr, _ := gjson.EncodeString(msg)

	data := []byte(dataStr)

	manager.mu.RLock()

	clients := make([]*Client, 0, len(room.Clients))
	for _, c := range room.Clients {
		if user == nil || c.User.Id != user.Id {
			clients = append(clients, c)
		}
	}
	manager.mu.RUnlock()

	fmt.Println("broadcast type:", msg.Type)

	for _, c := range clients {
		if c.IsClosed() {
			continue
		}

		select {
		case c.send <- data:
		default:
			g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
		}
	}
}

// 单发消息
func (manager *Manager) unicastAsync(ctx context.Context, c *Client, msg WSMessage) {
	if c == nil || c.IsClosed() {
		g.Log("test").Async().Errorf(ctx, "发消息失败，客户端是 nil: %v, 客户端 closed: %v", c == nil, c.IsClosed())
		return
	}

	dataStr, _ := gjson.EncodeString(msg)

	fmt.Println("unicast type:", msg.Type)

	data := []byte(dataStr)
	select {
	case c.send <- data:
	default:
		g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
	}
}
