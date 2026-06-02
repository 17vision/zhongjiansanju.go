package websocket

// WebSocket消息类型常量
// 用于前后端通信的 type 字段，所有消息类型都应集中定义，避免魔法字符串
const (
	MsgTypePing        = "ping"         // 心跳包
	MsgTypePong        = "pong"         // 心跳响应
	MsgTypeJoined      = "joined"       // 加入房间成功
	MsgTypeMoved       = "moved"        // 切换房间/地图
	MsgTypeKicked      = "kicked"       // 被踢出房间
	MsgTypeKickedRoom  = "kicked_room"  // 被踢出房间
	MsgTypeRoomUsers   = "room_users"   // 房间用户列表
	MsgTypeUserJoined  = "user_joined"  // 有用户加入房间
	MsgTypeUserLeft    = "user_left"    // 有用户离开房间
	MsgTypeError       = "error"        // 错误消息
	MsgTypeStartGame   = "start_game"   // 游戏开始
	MsgTypeStopGame    = "stop_game"    // 游戏结束
	MsgTypeUserIsReady = "userIsReady"  // 用户开始
	CanEnterRoom       = "canEnterRoom" // 可以进入房间
)
