package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
)

type Scene struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Manager struct {
	mu        sync.RWMutex
	clients   map[uint64]*Client
	waitUsers []*User
	room      *Room
	userLocks sync.Map
	lastId    uint64
}

var mapSeq int64
var lobbySeq int64

func NewManager(capacity int) *Manager {
	files, err := gfile.ScanDir(gfile.Join(gfile.Pwd(), "storage/config"), "*.json", false) // false 不递归
	if err == nil {
		for _, f := range files {
			sceneName := gfile.Name(f)
			content := gfile.GetContents(f)
			fmt.Println(sceneName)
			fmt.Println(content)
		}
	}

	room := &Room{Id: "room", Name: "我的房间", Capacity: capacity, Clients: make(map[uint64]*Client), SceneIndex: -1}

	return &Manager{
		clients:   make(map[uint64]*Client),
		room:      room,
		waitUsers: make([]*User, 0),
		lastId:    0,
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

	// 这个用户如果已经在 clients 中
	oldClient, hadOld := manager.clients[user.Id]
	var oldRoom *Room
	if hadOld && oldClient != nil {
		if _, exists := manager.room.Clients[user.Id]; exists {
			delete(manager.room.Clients, user.Id)
			oldRoom = manager.room
		}
		delete(manager.clients, user.Id)
	}

	user.Extend.ConnectTime = time.Now().Unix()

	manager.waitUsers = append(manager.waitUsers, user)

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

func (manager *Manager) joinRoom(ctx context.Context) {
	manager.mu.Lock()

	if len(manager.waitUsers) == 0 {
		manager.mu.Unlock()
		return
	}

	if len(manager.room.Clients) >= manager.room.Capacity {
		manager.mu.Unlock()
		return
	}

	user := manager.waitUsers[0]
	manager.waitUsers = manager.waitUsers[1:]

	client, ok := manager.clients[user.Id]
	if !ok || client == nil {
		g.Log("test").Async().Infof(ctx, "不应该的错误，client 不存在，用户 id: %d", user.Id)
		manager.mu.Unlock()
		return
	}
	manager.room.Clients[client.User.Id] = client

	// 房间里的所有用户
	users := make([]*User, 0, len(manager.room.Clients))
	for _, c := range manager.room.Clients {
		if c.User != nil {
			users = append(users, c.User)
		}
	}

	manager.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeRoomUsers,
		Data: RoomUsersData{RoomId: manager.room.Id, Users: users},
	})

	manager.mu.Unlock()

	// 广播
	manager.broadcastAsync(ctx, manager.room, WSMessage{
		Type: MsgTypeUserJoined,
		Data: UserEventData{RoomId: manager.room.Id, User: user},
	}, user)
}

func (manager *Manager) start(ctx context.Context, uids []string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	var clients []*Client
	for _, item := range uids {
		userId := gconv.Uint64(item)

		client, ok := manager.room.Clients[userId]
		if ok && client != nil && !client.User.Extend.IsStart {
			clients = append(clients, client)
		}
	}

	if len(clients) == 0 {
		return false
	}

	manager.mu.Unlock()

	now := time.Now().Unix()
	for _, client := range clients {
		client.User.Extend.IsStart = true
		client.User.Extend.StartTime = now
	}

	go func() {
		for _, client := range clients {
			manager.unicastAsync(ctx, client, WSMessage{
				Type: MsgTypeUserStart,
				Data: client.User.Id,
			})
		}
	}()

	return true
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

	for index, user := range manager.waitUsers {
		if user.Id == userId {
			copy(manager.waitUsers[index:], manager.waitUsers[index+1:])
			manager.waitUsers = manager.waitUsers[:len(manager.waitUsers)-1]
		}
	}

	delete(manager.room.Clients, userId)

	manager.mu.Unlock()
	manager.broadcastUserLeft(ctx, manager.room, client.User)
	manager.joinRoom(ctx)
}

func (manager *Manager) error(ctx context.Context, client *Client, err error) {
	manager.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeError,
		Data: err.Error(),
	})
}

// 广播消息
func (manager *Manager) broadcastAsync(ctx context.Context, room *Room, msg WSMessage, user *User) {
	dataStr, err := gjson.EncodeString(msg)
	if err != nil {
		g.Log("test").Async().Infof(ctx, "fun broadcastAsync -> json encode error -> message: %v", msg.Data)
		return
	}
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

	dataStr, err := gjson.EncodeString(msg)
	if err != nil {
		g.Log("test").Async().Infof(ctx, "fun unicastAsync -> json encode error -> message: %v", msg.Data)
		return
	}
	data := []byte(dataStr)

	select {
	case c.send <- data:
	default:
		g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
	}
}
