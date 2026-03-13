package websocket

// WebSocket消息类型常量
// 用于前后端通信的 type 字段，所有消息类型都应集中定义，避免魔法字符串
const (
	MsgTypePing         = "ping"              // 心跳包
	MsgTypePong         = "pong"              // 心跳响应
	MsgTypeJoined       = "joined"            // 加入房间成功
	MsgTypeMoved        = "moved"             // 切换房间/地图
	MsgTypeRoomUsers    = "room_users"        // 房间用户列表
	MsgTypeUserJoined   = "user_joined"       // 有用户加入房间
	MsgTypeUserLeft     = "user_left"         // 有用户离开房间
	MsgTypeAddFriend    = "add_friend"        // 加好友请求
	MsgTypeFriendAdded  = "friend_added"      // 好友添加成功通知
	MsgTypeAddFriendRes = "add_friend_result" // 加好友结果
	MsgTypeError        = "error"             // 错误消息
	MsgTypeKicked       = "kicked"            // 被踢下线通知
	MsgTypeKickedRoom   = "kicked_romm"       // 踢出房间通知
	MsgTypeStartGame    = "start_game"        // 游戏开始
	MsgTypeUserIsReady  = "userIsReady"       // 用户准备
	MsgTypeUserStart    = "userStart"         // 用户开始
	MsgTypeUserStop     = "userStop"          // 用户结束
)

// 错误码常量
// 所有错误码都应集中定义，便于前后端约定和维护
const (
	ErrInvalidFriendID  = 1001 // 好友ID无效
	ErrFriendOffline    = 1002 // 好友不在线
	ErrRoomNotFound     = 1003 // 房间不存在
	ErrReconnectTooMany = 1004 // 重连次数过多
	ErrHeartbeatTimeout = 1005 // 心跳超时
)
