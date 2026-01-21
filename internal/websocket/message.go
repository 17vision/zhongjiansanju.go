package websocket

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
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
	RoomId    string `json:"roomId"`    // 房间ID
	User      *User  `json:"user"`      // 用户信息
	PosConfig string `json:"posConfig"` //位置配置信息
	Scenes    string `json:"scenes"`    // 场景配置
}

type Envelope struct {
	Type string      `json:"type"` // 消息类型
	Data *gjson.Json `json:"data"` // 延迟解析的原始 JSON
}

type ReadMessageHandlerFunc func(ctx context.Context, m *Manager, client *Client, message interface{}) error

var ReadMessageHandlers = map[string]ReadMessageHandlerFunc{
	"chat":            ChatHandler,
	"createOrJoinMap": CreateOrJoinMapHandler,
	"joinRoom":        JoinRoomHandler,
	"action":          ActionHandler,
	"userIsReady":     UserReadyHandler,
}

func ChatHandler(ctx context.Context, manager *Manager, client *Client, message interface{}) error {
	req := message.(*ChatReq)

	for _, client := range manager.room.Clients {
		if client.User.Id == req.ToUid || client.User.Id == req.FromUid {
			msg := WSMessage{
				Type: "chat",
				Data: req,
			}
			manager.unicastAsync(ctx, client, msg)
		}
	}
	return nil
}

func CreateOrJoinMapHandler(ctx context.Context, manager *Manager, client *Client, message interface{}) error {
	// req := message.(*CreateOrJoinMapReq)

	// // 先退出以前的房子
	// manager.leftLobbyOrMap(ctx, client)

	// // 再创建或进房子
	// room := manager.createOrJoinMap(client, req.Map)

	// g.Log("test").Async().Infof(ctx, "用户 %s 创建或进入地图 %s", client.User.Nickname, room.Id)

	// var scenes []byte
	// if room.MapBase != "" {
	// 	roomScenes := manager.scenes[room.MapBase]
	// 	if roomScenes != nil {
	// 		scenes, _ = gjson.Marshal(roomScenes)
	// 	}
	// }

	// // 发给自己，加入了房间
	// manager.unicastAsync(ctx, client, WSMessage{
	// 	Type: MsgTypeJoined,
	// 	Data: UserEventData{RoomId: room.Id, User: client.User, Scenes: string(scenes), PosConfig: manager.posJson},
	// })

	// // 把房间里的人推送给自己
	// manager.pushRoomUserList(ctx, room, client.User.Id)

	// // 广播消息，有人进来了
	// manager.broadcastUserJoined(ctx, room, client.User)

	return nil
}

func JoinRoomHandler(ctx context.Context, manager *Manager, client *Client, message interface{}) error {
	// req := message.(*JoinRoomReq)

	// err := manager.JoinRoom(ctx, client.User.Id, req.RoomId)

	// if err != nil {
	// 	manager.error(ctx, client, err)
	// }
	return nil
}

func ActionHandler(ctx context.Context, manager *Manager, client *Client, message interface{}) error {
	req := message.(*ActionReq)

	g.Log("test").Async().Infof(ctx, "%s 发起 Action = %s", client.User.Nickname, req.Action)

	// if req.Action == "changeScene" {
	// 	result := manager.changeScene(ctx, client.RoomId, gconv.Int(req.Data))
	// 	g.Log("test").Async().Infof(ctx, "用户 %d changeScene %d %v ", client.User.Id, gconv.Int(req.Data), result)
	// }

	return nil
}

func UserReadyHandler(ctx context.Context, manager *Manager, client *Client, message interface{}) error {
	// client.User.Extend.IsReady = true
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

type CreateOrJoinMapReq struct {
	Map string `json:"map"`
}

type JoinRoomReq struct {
	RoomId string `json:"roomId"`
}

// 通用行为定义
type ActionReq struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}
