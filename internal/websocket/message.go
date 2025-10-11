package websocket

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
