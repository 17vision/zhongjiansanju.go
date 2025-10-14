package websocket

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/encoding/gjson"
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

func (m *Manager) KickClient(ctx context.Context, user *User, reason string) {
	m.mu.Lock()
	client, ok := m.clients[user.Id]
	m.mu.Unlock()

	if !ok || client == nil {
		return
	}

	g.Log("test").Async().Infof(ctx, "%s 被踢下线通知", client.User.Nickname)

	// 先发通知，再踢出去
	m.unicastAsync(ctx, client, WSMessage{
		Type: MsgTypeKicked,
		Data: map[string]any{"reason": reason},
	})

	m.RemoveClient(ctx, user.Id)
}

// CreateOrJoinLobby finds a lobby with space or creates one
func (m *Manager) CreateOrJoinLobby(user *User, conn *websocket.Conn) (*Room, *Client) {
	// 异地登录
	m.KickClient(context.Background(), user, "异地登录")

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, room := range m.rooms[RoomTypeLobby] {
		if len(room.Clients) < m.lobbyCapacity {
			client := &Client{User: user, RoomType: RoomTypeLobby, RoomId: room.Id, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}
			room.Clients[user.Id] = client
			m.clients[user.Id] = client
			return room, client
		}
	}

	rid := fmt.Sprintf("lobby-%d", len(m.rooms[RoomTypeLobby])+1)
	room := &Room{Id: rid, Name: "大厅", Type: RoomTypeLobby, Clients: make(map[uint64]*Client)}
	client := &Client{User: user, RoomType: RoomTypeLobby, RoomId: room.Id, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}
	m.rooms[RoomTypeLobby] = append(m.rooms[RoomTypeLobby], room)
	m.clients[user.Id] = client
	room.Clients[user.Id] = client
	return room, client
}

// CreateOrJoinMap finds a map room with space or creates one
// func (m *Manager) CreateOrJoinMap(user *User, mapBase string) (*Room, error) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	// try to find existing map rooms with space
// 	for _, r := range m.rooms[RoomTypeMap] {
// 		if r.Id == mapBase || len(r.Clients) < m.mapCapacity {
// 			client := &Client{User: user, RoomId: r.Id}
// 			r.Clients[user.Id] = client
// 			m.clients[user.Id] = client
// 			return r, nil
// 		}
// 	}
// 	// create new map instance
// 	rid := fmt.Sprintf("%s-%d", mapBase, len(m.rooms[RoomTypeMap])+1)
// 	r := &Room{Id: rid, Type: RoomTypeMap, Clients: make(map[uint64]*Client)}
// 	client := &Client{User: user, RoomId: r.Id}
// 	r.Clients[user.Id] = client
// 	m.rooms[RoomTypeMap] = append(m.rooms[RoomTypeMap], r)
// 	m.clients[user.Id] = client
// 	return r, nil
// }

// JoinRoom moves a user to specified room (used for teleport to friend's map)
// func (m *Manager) JoinRoom(userId uint64, roomID string) error {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	client, ok := m.clients[userId]
// 	if !ok {
// 		return fmt.Errorf("user not found")
// 	}
// 	// leave old room
// 	for rt, list := range m.rooms {
// 		for _, r := range list {
// 			if _, exists := r.Clients[userId]; exists {
// 				delete(r.Clients, userId)
// 				break
// 			}
// 		}
// 		_ = rt // suppress unused
// 	}
// 	// find target
// 	for _, list := range m.rooms {
// 		for _, r := range list {
// 			if r.Id == roomID {
// 				r.Clients[userId] = client
// 				client.RoomId = r.Id
// 				return nil
// 			}
// 		}
// 	}
// 	return fmt.Errorf("room not found")
// }

// PushRoomUserListTo 只推送房间用户列表给指定用户
func (m *Manager) PushRoomUserList(ctx context.Context, room *Room, userId uint64) {
	users := make([]*User, 0, len(room.Clients))
	for _, c := range room.Clients {
		if c.User != nil {
			users = append(users, c.User)
		}
	}

	if c, ok := room.Clients[userId]; ok {
		m.unicastAsync(ctx, c, WSMessage{
			Type: MsgTypeRoomUsers,
			Data: RoomUsersData{RoomId: room.Id, Users: users},
		})
	}
}

// 广播新用户加入
func (m *Manager) BroadcastUserJoined(ctx context.Context, room *Room, user *User) {
	m.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeUserJoined,
		Data: UserEventData{RoomId: room.Id, User: user},
	}, user)
}

// 广播用户离开房间
func (m *Manager) BroadcastUserLeft(ctx context.Context, room *Room, user *User) {
	m.broadcastAsync(ctx, room, WSMessage{
		Type: MsgTypeUserLeft,
		Data: UserEventData{RoomId: room.Id, User: user},
	}, user)
}

// 移除 client
func (m *Manager) RemoveClient(ctx context.Context, userId uint64) {
	m.mu.Lock()
	client, ok := m.clients[userId]
	if !ok {
		m.mu.Unlock()
		return
	}

	g.Log("test").Async().Infof(ctx, "用户 %s 退出", client.User.Nickname)

	delete(m.clients, userId)

	var room *Room
	for _, list := range m.rooms {
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
	m.mu.Unlock()

	if room != nil {
		m.broadcastAsync(ctx, room, WSMessage{
			Type: MsgTypeUserLeft,
			Data: UserEventData{RoomId: room.Id, User: client.User},
		}, nil)
	}
}

// 广播消息
func (m *Manager) broadcastAsync(ctx context.Context, room *Room, msg WSMessage, user *User) {
	dataStr, _ := gjson.EncodeString(msg)
	data := []byte(dataStr)

	m.mu.RLock()

	clients := make([]*Client, 0, len(room.Clients))
	for _, c := range room.Clients {
		if user == nil || c.User.Id != user.Id {
			clients = append(clients, c)
		}
	}
	m.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- data:
		default:
			g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
		}
	}
}

// 单发消息
func (m *Manager) unicastAsync(ctx context.Context, c *Client, msg WSMessage) {
	dataStr, _ := gjson.EncodeString(msg)
	data := []byte(dataStr)
	select {
	case c.send <- data:
	default:
		g.Log("test").Async().Infof(ctx, "client %d send chan full, skip", c.User.Id)
	}
}
