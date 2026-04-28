package websocket

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
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
var recordHost string = "https://game.17vision.com"

func NewManager(lobbyCap, mapCap int) *Manager {
	var gameConfig *GameConfig
	gameConfigPath := gfile.Join(gfile.Pwd(), "storage/config", "game.json")
	gameConfigJson := gfile.GetContents(gameConfigPath)
	if gameConfigJson == "" {
		g.Log("test").Async().Errorf(context.TODO(), "配置文件不存在: %s", gameConfigPath)
		panic("缺少配置文件")
	} else {
		g.Log("test").Async().Infof(context.TODO(), "配置文件: %s", gameConfigPath)
	}

	gjson.Unmarshal([]byte(gameConfigJson), &gameConfig)

	if gameConfig.RecordHost != "" {
		recordHost = gameConfig.RecordHost
	}

	return &Manager{
		rooms:             make(map[RoomType][]*Room),
		clients:           make(map[uint64]*Client),
		gameServerClients: make(map[uint64]*Client),
		userLocks:         sync.Map{},
		gameConfig:        gameConfig,
	}
}

// 创建一个 Client
func (manager *Manager) createClient(ctx context.Context, user *User, conn *websocket.Conn) *Client {
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
	client := &Client{User: user, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}
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

	delete(room.Clients, user.Id)

	// 广播用户离开房间
	manager.broadcastUserLeft(ctx, room, user)

	if len(room.Clients) == 0 {
		manager.destroyRoom()
	}

	// 进入或创建房间
	room = manager.createOrJoinLobby(client)

	// 把房间里的人推送给自己
	manager.pushRoomUserList(ctx, room, user.Id)

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

	room := &Room{Id: roomId, Name: "大厅", Type: RoomTypeLobby, Capacity: manager.gameConfig.LobbyCapacity, Clients: make(map[uint64]*Client)}
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
func (manager *Manager) createOrJoinMap(client *Client, mapBase string) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client.RoomType = RoomTypeMap

	mapCapacity := manager.gameConfig.MapCapacity

	for _, room := range manager.rooms[RoomTypeMap] {
		// 房间是 wating 状态,并且人数小于设定人数,才可以进(假如存在多个没满,当前逻辑不存在.就应该可以指定房间进的概念)
		if room.MapBase == mapBase && room.Status == RoomStatusWating && len(room.Clients) < mapCapacity {
			client.RoomId = room.Id
			room.Clients[client.User.Id] = client
			return room
		}
	}

	var selectClient *Client
	for _, tempClient := range manager.gameServerClients {
		if tempClient.User.Extend.IsServer == false {
			selectClient = tempClient
			break
		}
	}

	if selectClient == nil {
		return nil
	}
	selectClient.User.Extend.IsServer = true

	rid := fmt.Sprintf("%s-%d", mapBase, atomic.AddInt64(&mapSeq, 1))
	room := &Room{Id: rid, MapBase: mapBase, Type: RoomTypeMap, Name: "地图", Capacity: mapCapacity, Clients: make(map[uint64]*Client), Status: RoomStatusWating}
	room.GameServerClient = selectClient

	// 把绑定的房间 id 也带上
	selectClient.RoomId = room.Id

	client.RoomId = room.Id
	room.Clients[client.User.Id] = client
	manager.rooms[RoomTypeMap] = append(manager.rooms[RoomTypeMap], room)
	return room
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
			manager.destroyRoom()
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
	manager.mu.Unlock()

	var newRoom *Room
	for _, list := range manager.rooms {
		for _, r := range list {
			if r.Id == roomID {
				newRoom = r
				break
			}
		}
	}

	if newRoom == nil {
		return gerror.New("房间不存在[]")
	}

	var max int
	if newRoom.Type == RoomTypeLobby {
		max = manager.gameConfig.LobbyCapacity
	} else {
		max = manager.gameConfig.MapCapacity
	}

	if len(newRoom.Clients) >= max {
		return gerror.New("房间已满员")
	}

	// 先离开房间
	manager.leftLobbyOrMap(ctx, client)

	// 再加入房间
	newRoom.Clients[client.User.Id] = client

	// 把房间里的人推送给自己
	manager.pushRoomUserList(ctx, newRoom, client.User.Id)

	// 广播消息，有人进来了
	manager.broadcastUserJoined(ctx, newRoom, client.User)

	return fmt.Errorf("room not found")
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

	// 修改房间状态并记录时间（在锁内完成）
	room.Status = RoomStatusPlaying
	room.StartTime = time.Now().Unix()

	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "开始游戏, 房间 id = %s, 新状态 = %v", roomId, RoomStatusPlaying)

	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeStartGame,
		Data: nil,
	}, nil)

	// 通知游戏服务器开始游戏
	var sn string
	if room.GameServerClient != nil {
		manager.unicastAsync(ctx, room.GameServerClient, WSMessage{
			Type: MsgTypeStartGame,
			Data: nil,
		})

		sn = "start_success"
		g.Log("test").Async().Infof(ctx, "开始游戏 给游戏客户端发消息。用户 id = %d, 端口 = %d, ", room.GameServerClient.User.Id, room.GameServerClient.User.Port)
	} else {
		sn = "start_fail"
		g.Log("test").Async().Error(ctx, "开始游戏失败, 游戏客户端不存在")
	}

	// 设置房间用户状态，提交服务端统计
	for _, client := range room.Clients {
		client.User.Extend.StartTime = time.Now().Unix()

		data := g.Map{
			"name":       "hxnc-hongkou",
			"ip":         client.User.Extend.ConnectIp,
			"connect_at": gtime.NewFromTimeStamp(client.User.Extend.ConnectTime).Format("Y-m-d H:i:s"),
			"start_at":   gtime.NewFromTimeStamp(client.User.Extend.StartTime).Format("Y-m-d H:i:s"),
			"sn":         sn,
		}

		r, err := g.Client().Post(ctx, gstr.Join([]string{recordHost, "api/game/start_records"}, "/"), data)

		if err != nil {
			continue
		}
		defer r.Close()

		result := r.ReadAllString()

		g.Log("test").Async().Infof(ctx, "提交统计返回数据:%s", result)

		type GameStartRecord struct {
			ID        uint64 `json:"id"`
			Name      string `json:"name"`
			ConnectAt string `json:"connect_at"`
			StartAt   string `json:"start_at"`
		}

		var gameStartRecord GameStartRecord
		err = gjson.Unmarshal([]byte(result), &gameStartRecord)
		if err == nil {
			client.User.Extend.RecordId = gameStartRecord.ID
			g.Log("test").Async().Infof(ctx, "用户 %d 开始游戏已统计。统计 id 是 %d", client.User.Id, gameStartRecord.ID)
		} else {
			g.Log("test").Async().Errorf(ctx, "用户 %d 开始游戏统计失败。错误是 %s", client.User.Id, err.Error())
		}
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

	// 🔴 关键：拷贝 clients 引用，避免解锁后遍历原 map
	clients := make([]*Client, 0, len(room.Clients))
	for _, c := range room.Clients {
		clients = append(clients, c)
	}

	manager.mu.Unlock()

	g.Log("test").Async().Infof(ctx, "结束游戏, 房间 id = %s, 新状态 = %v", roomId, RoomStatusStopped)

	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeStopGame,
		Data: nil,
	}, nil)

	if room.GameServerClient != nil {
		manager.unicastAsync(ctx, room.GameServerClient, WSMessage{
			Type: MsgTypeStopGame,
			Data: nil,
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

// 推送房间用户
func (manager *Manager) pushRoomUserList(ctx context.Context, room *Room, userId uint64) {
	users := make([]*User, 0, len(room.Clients))

	for _, c := range room.Clients {
		if c.User != nil {
			users = append(users, c.User)
		}
	}

	if c, ok := room.Clients[userId]; ok {
		manager.unicastAsync(ctx, c, WSMessage{
			Type: MsgTypeRoomUsers,
			Data: RoomUsersData{RoomId: room.Id, Users: users},
		})
	}
}

// 广播新用户加入
func (manager *Manager) broadcastUserJoined(ctx context.Context, room *Room, user *User) {
	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeUserJoined,
		Data: UserEventData{RoomId: room.Id, UserId: user.Id, Ip: manager.gameConfig.Ip},
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
	manager.mu.Lock()
	client, ok := manager.clients[userId]
	if !ok {
		manager.mu.Unlock()
		return
	}

	g.Log("test").Async().Infof(ctx, "用户 %d 退出", client.User.Id)

	delete(manager.clients, userId)

	recordId := client.User.Extend.RecordId

	// 如果是游戏服务器挂了，就删除游戏服务器
	if client.User.Type == TypeGameServer {
		client.User.Extend.IsServer = false
		client.RoomId = ""
		// 删除引用
		delete(manager.gameServerClients, userId)
		g.Log("test").Async().Infof(ctx, "游戏服务器退出，删除引用")
	}

	var room *Room
	for _, list := range manager.rooms {
		for _, r := range list {
			if _, exists := r.Clients[userId]; exists {
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

	if room != nil {
		manager.broadcastUserLeft(ctx, room, client.User)

		if len(room.Clients) == 0 {
			g.Log("test").Async().Infof(ctx, "房间无人，销毁房间 %s ", room.Id)
			if room.GameServerClient != nil {
				room.GameServerClient.User.Extend.IsServer = false
				room.GameServerClient.RoomId = ""
				room.GameServerClient = nil

				g.Log("test").Async().Infof(ctx, "销毁房间，复位数据:")
				g.Log("test").Async().Infof(ctx, gjson.MustEncodeString(manager.gameServerClients))
			}

			manager.destroyRoom()
		}
	}

	if recordId != 0 {
		apiCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := g.Client().Put(apiCtx, gstr.Join([]string{recordHost, "api/game/start_records"}, "/"), g.Map{"id": recordId, "end_at": gtime.NewFromTimeStamp(time.Now().Unix()).Format("Y-m-d H:i:s")})

		if err != nil {
			g.Log("test").Errorf(apiCtx, "上报结束失败 recordId=%d err=%v", recordId, err)
		} else {
			g.Log("test").Debugf(apiCtx, "上报结束成功 recordId=%d resp=%s", recordId, res.ReadAllString())
		}
	}
}

func (manager *Manager) destroyRoom() {
	manager.mu.Lock()

	defer manager.mu.Unlock()

	for index, list := range manager.rooms {
		filtered := list[:0]
		for _, r := range list {
			if len(r.Clients) > 0 {
				filtered = append(filtered, r)
			}
		}
		manager.rooms[index] = filtered
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

	data := []byte(dataStr)
	select {
	case c.send <- data:
	default:
		g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
	}
}
