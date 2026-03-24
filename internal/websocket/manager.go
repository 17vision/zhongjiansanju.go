package websocket

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"zjsj/internal/pkg/utils"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gorilla/websocket"
)

type Scene struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MapConfig struct {
	name     string
	mapBase  string
	capacity int
}

type Manager struct {
	mu            sync.RWMutex
	rooms         map[RoomType][]*Room
	clients       map[uint64]*Client
	scenes        map[string][]*Scene
	mapConfigs    map[string]*MapConfig
	lobbyCapacity int
	mapCapacity   int
	userLocks     sync.Map

	// 活着的用户
	liveUsers map[string]*User
	lastId    uint64
	posJson   string
}

var mapSeq int64
var lobbySeq int64

func NewManager(lobbyCap, mapCap int) *Manager {
	// 读取 posJson 配置文件
	posPath := gfile.Join(gfile.Pwd(), "storage", "posJson.json")
	posJson := gfile.GetContents(posPath)

	// 读取
	var mapConfigs map[string]*MapConfig = make(map[string]*MapConfig)
	mapPath := gfile.Join(gfile.Pwd(), "storage", "mapConfig.json")
	mapConfig := gfile.GetContents(mapPath)
	if mapConfig != "" {
		gjson.Unmarshal([]byte(mapConfig), &mapConfigs)
	}

	// 读取场景配置文件
	scenes := make(map[string][]*Scene)
	files, err := gfile.ScanDir(gfile.Join(gfile.Pwd(), "storage/scenes"), "*.json", false) // false 不递归
	if err == nil {
		for _, f := range files {
			sceneName := gfile.Name(f)
			content := gfile.GetContents(f)
			var tempScenes []*Scene
			if err = gjson.Unmarshal([]byte(content), &tempScenes); err == nil {
				scenes[sceneName] = tempScenes
			}
		}
	}

	return &Manager{
		clients:       make(map[uint64]*Client),
		rooms:         make(map[RoomType][]*Room),
		scenes:        scenes,
		mapConfigs:    mapConfigs,
		lobbyCapacity: lobbyCap,
		mapCapacity:   mapCap,
		liveUsers:     make(map[string]*User),
		lastId:        0,
		posJson:       posJson,
	}
}

func (manager *Manager) getUser(device_id string) *User {
	user, had := manager.liveUsers[device_id]
	if had && user != nil {
		return user
	}

	manager.mu.Lock()

	var getName func() utils.NameResult
	getName = func() utils.NameResult {
		n := utils.RandomName()

		for _, u := range manager.liveUsers {
			if u != nil && u.Nickname == n.Name {
				return getName()
			}
		}
		return n
	}

	name := getName()

	user = &User{
		Id:       manager.lastId + 1,
		Nickname: name.Name,
		Gender:   Gender(name.Gender),
		Avatar:   name.Avatar,
	}

	manager.liveUsers[device_id] = user
	manager.lastId++
	manager.mu.Unlock()
	return user
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
		g.Log("test").Async().Infof(ctx, "%s 被踢下线通知", oldClient.User.Nickname)

		// 先发通知
		manager.unicastAsync(ctx, oldClient, WSMessage{
			Type: MsgTypeKicked,
			Data: map[string]any{"reason": "异地登录"},
		})

		// 广播旧用户离开旧房间（如果有）
		if oldRoom != nil {
			manager.broadcastAsync(ctx, oldRoom, WSMessage{
				Type: MsgTypeUserLeft,
				Data: UserEventData{RoomId: oldRoom.Id, User: oldClient.User},
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

	g.Log("test").Async().Infof(ctx, "%s 被踢下线通知", client.User.Nickname)

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

	g.Log("test").Async().Infof(ctx, "%s 被踢出房间通知", client.User.Nickname)

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

// 创建或加入一个大厅
func (manager *Manager) createOrJoinLobby(client *Client) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client.RoomType = RoomTypeLobby

	for _, room := range manager.rooms[RoomTypeLobby] {
		if len(room.Clients) < manager.lobbyCapacity {
			client.RoomId = room.Id
			room.Clients[client.User.Id] = client
			return room
		}
	}

	// roomId := fmt.Sprintf("lobby-%d", len(manager.rooms[RoomTypeLobby])+1)
	roomId := fmt.Sprintf("lobby-%d", atomic.AddInt64(&lobbySeq, 1))

	room := &Room{Id: roomId, Name: "大厅", Type: RoomTypeLobby, Capacity: manager.lobbyCapacity, Clients: make(map[uint64]*Client), SceneIndex: -1}
	client.RoomId = room.Id
	room.Clients[client.User.Id] = client
	manager.rooms[RoomTypeLobby] = append(manager.rooms[RoomTypeLobby], room)

	// 将场景绑定到房间上边
	scenes, ok := manager.scenes[room.MapBase]
	if ok {
		room.Scenes = scenes
	}

	return room
}

// 创建或进入地图【大厅只有一个，地图却有很多，需要通过 mapBase 来区分】
func (manager *Manager) createOrJoinMap(client *Client, mapBase string) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client.RoomType = RoomTypeMap

	mapCapacity := manager.mapCapacity
	mapConfig := manager.mapConfigs[mapBase]
	if mapConfig != nil {
		mapCapacity = mapConfig.capacity
	}

	for _, room := range manager.rooms[RoomTypeMap] {
		// 房间是 wating 状态,并且人数小于设定人数,才可以进(假如存在多个没满,当前逻辑不存在.就应该可以指定房间进的概念)
		if room.MapBase == mapBase && room.Status == RoomStatusWating && len(room.Clients) < mapCapacity {
			client.RoomId = room.Id
			room.Clients[client.User.Id] = client

			// 分配名字
			manager.assignNames(room, client.User)
			return room
		}
	}

	// create new map instance
	// rid := fmt.Sprintf("%s-%d", mapBase, len(manager.rooms[RoomTypeMap])+1)
	rid := fmt.Sprintf("%s-%d", mapBase, atomic.AddInt64(&mapSeq, 1))
	room := &Room{Id: rid, MapBase: mapBase, Type: RoomTypeMap, Name: "地图", Capacity: mapCapacity, Clients: make(map[uint64]*Client), Status: RoomStatusWating, Usernames: append([]string(nil), Room_Usernames...), SceneIndex: -1}

	client.RoomId = room.Id
	room.Clients[client.User.Id] = client
	manager.rooms[RoomTypeMap] = append(manager.rooms[RoomTypeMap], room)

	// 将场景绑定到房间上边
	scenes, ok := manager.scenes[room.MapBase]
	if ok {
		room.Scenes = scenes
	}

	// 分配名字
	manager.assignNames(room, client.User)
	return room
}

// 分配名字
func (manager *Manager) assignNames(room *Room, user *User) bool {
	if len(room.Usernames) == 0 {
		return false
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	index := r.Intn(len(room.Usernames))

	username := room.Usernames[index]

	room.Usernames = append(room.Usernames[:index], room.Usernames[index+1:]...)

	user.Nickname = username
	user.Avatar = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", username)
	user.Extend.IsReady = false
	return true
}

func (manager *Manager) recycleNames(room *Room, name string) bool {
	room.Usernames = append(room.Usernames, name)

	if len(room.Clients) == 0 {
		room.Status = RoomStatusWating
	}
	return true
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
		if room.Type == RoomTypeMap {
			manager.recycleNames(room, client.User.Nickname)
		}

		manager.broadcastUserLeft(ctx, room, client.User)

		if len(room.Clients) == 0 {
			manager.destroyRoom()
		}
	} else {
		g.Log("test").Async().Infof(ctx, "用户 %s 离开房间失败，找不到房间 %s", client.User.Nickname, client.RoomId)
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
		max = manager.lobbyCapacity
	} else {
		max = manager.mapCapacity
		mapConfig := manager.mapConfigs[newRoom.MapBase]
		if mapConfig != nil {
			max = mapConfig.capacity
		}
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

// 设置当前房间的场景
func (manager *Manager) changeScene(ctx context.Context, roomId string, index int) bool {
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
		g.Log("test").Async().Infof(ctx, "切换场景失败，没找到房间: %s", roomId)
		return false
	}
	manager.mu.Unlock()

	if index > room.SceneIndex && len(room.Scenes) >= (index+1) && index < len(room.Scenes) {
		room.SceneIndex = index
		return true
	}
	return false
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
		Data: UserEventData{RoomId: room.Id, User: user},
	}, user)
}

// 广播用户离开房间
func (manager *Manager) broadcastUserLeft(ctx context.Context, room *Room, user *User) {
	manager.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeUserLeft,
		Data: UserEventData{RoomId: room.Id, User: user},
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

	g.Log("test").Async().Infof(ctx, "用户 %s 退出", client.User.Nickname)

	delete(manager.clients, userId)

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
		// 回收名字
		if room.Type == RoomTypeMap {
			manager.recycleNames(room, client.User.Nickname)
		}
		manager.broadcastUserLeft(ctx, room, client.User)

		if len(room.Clients) == 0 {
			manager.destroyRoom()
		}
	}

	// if len(manager.clients) == 0 {
	// 	manager.lastId = 0
	// }
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
