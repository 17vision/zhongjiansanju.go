package websocket

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gorilla/websocket"
)

type Manager struct {
	mu            sync.RWMutex
	rooms         map[RoomType][]*Room
	clients       map[uint64]*Client
	lobbyCapacity int
	mapCapacity   int
}

func NewManager(lobbyCap, mapCap int) *Manager {
	return &Manager{
		clients:       make(map[uint64]*Client),
		rooms:         make(map[RoomType][]*Room),
		lobbyCapacity: lobbyCap,
		mapCapacity:   mapCap,
	}
}

// 创建一个 Client
func (manager *Manager) createClient(ctx context.Context, user *User, conn *websocket.Conn) *Client {
	// 异地登录
	manager.kickClient(ctx, user, "异地登录")

	manager.mu.Lock()

	client := &Client{User: user, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}

	manager.clients[user.Id] = client

	manager.mu.Unlock()

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

	roomId := fmt.Sprintf("lobby-%d", len(manager.rooms[RoomTypeLobby])+1)
	room := &Room{Id: roomId, Name: "大厅", Type: RoomTypeLobby, Clients: make(map[uint64]*Client)}
	client.RoomId = room.Id
	room.Clients[client.User.Id] = client
	manager.rooms[RoomTypeLobby] = append(manager.rooms[RoomTypeLobby], room)
	return room
}

// 创建或进入地图【大厅只有一个，地图却有很多，需要通过 mapBase 来区分】
func (manager *Manager) createOrJoinMap(client *Client, mapBase string) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client.RoomType = RoomTypeMap

	for _, room := range manager.rooms[RoomTypeMap] {
		if room.Id == mapBase || len(room.Clients) < manager.mapCapacity {
			client.RoomId = room.Id
			room.Clients[client.User.Id] = client
			return room
		}
	}

	// create new map instance
	rid := fmt.Sprintf("%s-%d", mapBase, len(manager.rooms[RoomTypeMap])+1)
	room := &Room{Id: rid, Type: RoomTypeMap, Name: "地图", Clients: make(map[uint64]*Client)}

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
		manager.broadcastUserLeft(ctx, room, client.User)
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
		select {
		case c.send <- data:
		default:
			g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
		}
	}
}

// 单发消息
func (manager *Manager) unicastAsync(ctx context.Context, c *Client, msg WSMessage) {
	dataStr, _ := gjson.EncodeString(msg)
	data := []byte(dataStr)
	select {
	case c.send <- data:
	default:
		g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
	}
}
