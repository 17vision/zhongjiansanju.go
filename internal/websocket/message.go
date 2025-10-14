package websocket

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/encoding/gjson"
)

// WSMessage WebSocket消息通用结构体
// 所有消息收发都应使用该结构体，便于协议统一和扩展
// Type: 消息类型，Data: 业务数据，Code/Msg: 错误码和错误信息
// 例如：{"type":"room_users","data":{...}}
type WSMessage struct {
	Type string      `json:"type"`           // 消息类型
	Data interface{} `json:"data,omitempty"` // 业务数据
	Code int         `json:"code,omitempty"` // 错误码
	Msg  string      `json:"msg,omitempty"`  // 错误信息
}

// JoinedData 加入房间成功消息体
type JoinedData struct {
	RoomId string `json:"roomId"` // 房间ID
}

// MovedData 切换房间/地图消息体
type MovedData struct {
	RoomID string `json:"room_id"` // 新房间ID
}

// RoomUsersData 房间用户列表消息体
type RoomUsersData struct {
	RoomId string  `json:"roomId"` // 房间ID
	Users  []*User `json:"users"`  // 用户列表
}

// UserEventData 用户进出房间事件消息体
type UserEventData struct {
	RoomId string `json:"roomId"` // 房间ID
	User   *User  `json:"user"`   // 用户信息
}

type Envelope struct {
	Type string      `json:"type"` // 消息类型
	Data *gjson.Json `json:"data"` // 延迟解析的原始 JSON
}

type ReadMessageHandlerFunc func(ctx context.Context, m *Manager, client *Client, message interface{}) error

var ReadMessageHandlers = map[string]ReadMessageHandlerFunc{
	"chat": ChatHandler,
}

func ChatHandler(ctx context.Context, manager *Manager, client *Client, message interface{}) error {
	req := message.(*ChatReq)

	fmt.Println("谁发的消息", req.FromUid, req.ToUid, req.Message.Content, req.Message.Type)

	// manager.rooms[client.RoomType] is a slice of *Room (indexed by int), so range over that slice
	for _, room := range manager.rooms[client.RoomType] {
		for _, client := range room.Clients {
			if client.User.Id == req.ToUid || client.User.Id == req.FromUid {
				msg := WSMessage{
					Type: "chat",
					Data: req,
				}

				manager.unicastAsync(ctx, client, msg)
			}
		}
	}
	return nil
}

type ChatMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}
type ChatReq struct {
	FromUid uint64      `json:"fromUid"`
	ToUid   uint64      `json:"toUid"`
	Message ChatMessage `json:"message"`
}
