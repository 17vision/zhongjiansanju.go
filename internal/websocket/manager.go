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

// CreateOrJoinLobby finds a lobby with space or creates one
func (m *Manager) CreateOrJoinLobby(user *User) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// find lobby with space
	for _, r := range m.rooms[RoomTypeLobby] {
		if len(r.Clients) < m.lobbyCapacity {
			client := &Client{User: user, RoomId: r.Id}
			r.Clients[user.Id] = client
			m.clients[user.Id] = client
			return r, nil
		}
	}
	// create new
	rid := fmt.Sprintf("lobby-%d", len(m.rooms[RoomTypeLobby])+1)
	r := &Room{Id: rid, Type: RoomTypeLobby, Clients: make(map[uint64]*Client)}
	client := &Client{User: user, RoomId: r.Id}
	r.Clients[user.Id] = client
	m.rooms[RoomTypeLobby] = append(m.rooms[RoomTypeLobby], r)
	m.clients[user.Id] = client
	return r, nil
}

// CreateOrJoinMap finds a map room with space or creates one
func (m *Manager) CreateOrJoinMap(user *User, mapBase string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// try to find existing map rooms with space
	for _, r := range m.rooms[RoomTypeMap] {
		if r.Id == mapBase || len(r.Clients) < m.mapCapacity {
			client := &Client{User: user, RoomId: r.Id}
			r.Clients[user.Id] = client
			m.clients[user.Id] = client
			return r, nil
		}
	}
	// create new map instance
	rid := fmt.Sprintf("%s-%d", mapBase, len(m.rooms[RoomTypeMap])+1)
	r := &Room{Id: rid, Type: RoomTypeMap, Clients: make(map[uint64]*Client)}
	client := &Client{User: user, RoomId: r.Id}
	r.Clients[user.Id] = client
	m.rooms[RoomTypeMap] = append(m.rooms[RoomTypeMap], r)
	m.clients[user.Id] = client
	return r, nil
}

// JoinRoom moves a user to specified room (used for teleport to friend's map)
func (m *Manager) JoinRoom(userId uint64, roomID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[userId]
	if !ok {
		return fmt.Errorf("user not found")
	}
	// leave old room
	for rt, list := range m.rooms {
		for _, r := range list {
			if _, exists := r.Clients[userId]; exists {
				delete(r.Clients, userId)
				break
			}
		}
		_ = rt // suppress unused
	}
	// find target
	for _, list := range m.rooms {
		for _, r := range list {
			if r.Id == roomID {
				r.Clients[userId] = client
				client.RoomId = r.Id
				return nil
			}
		}
	}
	return fmt.Errorf("room not found")
}

// PushRoomUserList 向房间内所有在线用户推送用户列表
func (m *Manager) PushRoomUserList(ctx context.Context, room *Room) {
	users := make([]*User, 0, len(room.Clients))
	for _, c := range room.Clients {
		if c.User != nil {
			users = append(users, c.User)
		}
	}

	for _, c := range room.Clients {
		m.SendMessage(ctx, c.Conn, WSMessage{
			Type: MsgTypeRoomUsers,
			Data: RoomUsersData{RoomId: room.Id, Users: users},
		})
	}
}

// PushRoomUserListTo 只推送房间用户列表给指定用户
func (m *Manager) PushRoomUserListTo(ctx context.Context, room *Room, userId uint64) {
	users := make([]*User, 0, len(room.Clients))
	for _, c := range room.Clients {
		if c.User != nil {
			users = append(users, c.User)
		}
	}

	if c, ok := room.Clients[userId]; ok {
		m.SendMessage(ctx, c.Conn, WSMessage{
			Type: MsgTypeRoomUsers,
			Data: RoomUsersData{RoomId: room.Id, Users: users},
		})
	}
}

// BroadcastUserJoined 向房间所有用户广播新用户加入
func (m *Manager) BroadcastUserJoined(ctx context.Context, room *Room, user *User) {
	for _, c := range room.Clients {
		m.SendMessage(ctx, c.Conn, WSMessage{
			Type: MsgTypeUserJoined,
			Data: UserEventData{RoomId: room.Id, User: user},
		})
	}
}

// BroadcastUserJoinedExcept 向房间所有用户（除指定用户外）广播新用户加入
func (m *Manager) BroadcastUserJoinedExcept(ctx context.Context, room *Room, user *User) {
	for uid, c := range room.Clients {
		if uid == user.Id {
			continue
		}

		g.Log("test").Async().Infof(ctx, "给 %s 广播 %s 进入了房间 \n", c.User.Nickname, user.Nickname)

		m.SendMessage(ctx, c.Conn, WSMessage{
			Type: MsgTypeUserJoined,
			Data: UserEventData{RoomId: room.Id, User: user},
		})
	}
	g.Log("test").Async().Info(ctx, "--------------------------------\n\n")
}

// BroadcastUserLeft 向房间所有用户广播有用户离开
func (m *Manager) BroadcastUserLeft(ctx context.Context, room *Room, user *User) {
	for _, c := range room.Clients {
		if c.User.Id != user.Id {
			m.SendMessage(ctx, c.Conn, WSMessage{
				Type: MsgTypeUserLeft,
				Data: UserEventData{RoomId: room.Id, User: user},
			})
		}
	}
}

// RemoveClient cleans up a client: removes from its room, from clients map and broadcasts left
func (m *Manager) RemoveClient(ctx context.Context, userId uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[userId]
	if !ok {
		return
	}

	g.Log("test").Async().Infof(ctx, "用户 %s 退出", client.User.Nickname)

	// find and remove from room
	for _, list := range m.rooms {
		for _, r := range list {
			if _, exists := r.Clients[userId]; exists {
				// remove
				delete(r.Clients, userId)

				// broadcast left
				for _, c := range r.Clients {
					if c.Conn != nil {
						m.SendMessage(ctx, c.Conn, WSMessage{
							Type: MsgTypeUserLeft,
							Data: UserEventData{RoomId: r.Id, User: client.User},
						})
					}
				}
				break
			}
		}
	}

	delete(m.clients, userId)
}

func (m *Manager) SendMessage(ctx context.Context, conn *websocket.Conn, message WSMessage) bool {
	if conn == nil {
		g.Log("websocket").Error(ctx, "发送消息失败[0] ", "conn 已是 nil")
		return false
	}

	data, err := gjson.EncodeString(message)
	if err != nil {
		g.Log("websocket").Error(ctx, "发送消息失败[1] ", err)
		return false
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
		g.Log("websocket").Error(ctx, "发送消息失败[2] ", err)
		return false
	}
	return true
}
