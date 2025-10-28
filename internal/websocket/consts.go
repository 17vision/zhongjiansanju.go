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
	MsgTypeUserIsReady  = "userIsReady"       //用户开始
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

const posJson = `{"xin_shou_da_ting":{"OffsetX":-0.7369806,"OffsetY":0.0,"OffsetZ":-2.618712,"RotationX":0.0,"RotationY":275.293182,"RotationZ":0.0},"yu_zhou":{"OffsetX":0.503464162,"OffsetY":0.0,"OffsetZ":0.222047031,"RotationX":0.0,"RotationY":91.5971756,"RotationZ":0.0},"shan_ding_dong":{"OffsetX":0.4417247,"OffsetY":0.0,"OffsetZ":-0.5378442,"RotationX":0.0,"RotationY":88.5755844,"RotationZ":0.0},"he_mu_du+":{"OffsetX":-0.06428805,"OffsetY":0.0,"OffsetZ":-0.797242939,"RotationX":0.0,"RotationY":211.609924,"RotationZ":0.0},"chuan_yue":{"OffsetX":-1.17175984,"OffsetY":0.0,"OffsetZ":-0.765180767,"RotationX":0.0,"RotationY":271.674347,"RotationZ":0.0},"jing_zi_ta2":{"OffsetX":-0.287304372,"OffsetY":0.0,"OffsetZ":-0.7183295,"RotationX":0.0,"RotationY":273.979828,"RotationZ":0.0},"dou_shou_chang":{"OffsetX":-1.85395384,"OffsetY":0.0,"OffsetZ":0.156926334,"RotationX":0.0,"RotationY":301.311,"RotationZ":0.0},"shen_miao":{"OffsetX":-2.568809,"OffsetY":0.0,"OffsetZ":-0.528443635,"RotationX":0.0,"RotationY":87.49721,"RotationZ":0.0},"chang_cheng":{"OffsetX":-4.24604225,"OffsetY":0.0,"OffsetZ":-2.93165851,"RotationX":0.0,"RotationY":356.324432,"RotationZ":0.0},"tai_he_dian":{"OffsetX":-4.03623247,"OffsetY":0.0,"OffsetZ":-3.24233651,"RotationX":0.0,"RotationY":0.0,"RotationZ":0.0},"pan_zhi_hua":{"OffsetX":-7.08020163,"OffsetY":0.0,"OffsetZ":-0.6612325,"RotationX":0.0,"RotationY":65.82684,"RotationZ":0.0},"shen_zhen_guo_mao":{"OffsetX":-6.495438,"OffsetY":0.0,"OffsetZ":0.5746328,"RotationX":0.0,"RotationY":87.123085,"RotationZ":0.0},"shipin_DiWangDaSha":{"OffsetX":0.153955981,"OffsetY":0.0,"OffsetZ":-0.154660732,"RotationX":0.0,"RotationY":0.0,"RotationZ":0.0},"SH_jrzx":{"OffsetX":-1.858588,"OffsetY":0.0,"OffsetZ":-3.28560519,"RotationX":0.0,"RotationY":46.00231,"RotationZ":0.0},"shipin_ZhongDong":{"OffsetX":3.49163651,"OffsetY":0.0,"OffsetZ":6.29033232,"RotationX":0.0,"RotationY":302.4418,"RotationZ":0.0},"wei_lai_gong_di":{"OffsetX":2.33402276,"OffsetY":0.0,"OffsetZ":9.68956852,"RotationX":0.0,"RotationY":266.571136,"RotationZ":0.0},"yu_zhou2":{"OffsetX":2.25203562,"OffsetY":0.0,"OffsetZ":6.82328129,"RotationX":0.0,"RotationY":268.711884,"RotationZ":0.0},"HuoXing_JiDi":{"OffsetX":6.888966,"OffsetY":0.0,"OffsetZ":0.4452995,"RotationX":0.0,"RotationY":280.9117,"RotationZ":0.0},"zong_bu_chuan_yue":{"OffsetX":4.249589,"OffsetY":0.0,"OffsetZ":1.7196846,"RotationX":0.0,"RotationY":324.25174,"RotationZ":0.0},"XCZX":{"OffsetX":0.8156347,"OffsetY":0.0,"OffsetZ":45.4766769,"RotationX":0.0,"RotationY":262.820618,"RotationZ":0.0},"1_2_XingShouYinDao":{"OffsetX":-0.139579386,"OffsetY":0.0,"OffsetZ":-1.96474266,"RotationX":0.0,"RotationY":190.116074,"RotationZ":0.0},"6_HeMuDu":{"OffsetX":-0.214616925,"OffsetY":0.0,"OffsetZ":-4.838274,"RotationX":0.0,"RotationY":263.70755,"RotationZ":0.0},"7.1_JingZiTa":{"OffsetX":-1.15186048,"OffsetY":0.0,"OffsetZ":0.274946481,"RotationX":0.0,"RotationY":181.515549,"RotationZ":0.0},"7.3_ShenMiao":{"OffsetX":-1.87579286,"OffsetY":0.0,"OffsetZ":0.9127433,"RotationX":0.0,"RotationY":181.515549,"RotationZ":0.0},"8.1_ChangCheng":{"OffsetX":-4.905255,"OffsetY":0.0,"OffsetZ":-2.23383331,"RotationX":0.0,"RotationY":1.69521153,"RotationZ":0.0},"8.2_TaiHeDian":{"OffsetX":-4.120702,"OffsetY":0.0,"OffsetZ":-3.42686653,"RotationX":0.0,"RotationY":1.69521153,"RotationZ":0.0},"9.1_ShiShiDaHui":{"OffsetX":-1.838476,"OffsetY":0.0,"OffsetZ":1.59182453,"RotationX":0.0,"RotationY":169.723969,"RotationZ":0.0},"9.2_10_11_PanZhiHua":{"OffsetX":-3.592017,"OffsetY":0.0,"OffsetZ":-1.46183848,"RotationX":0.0,"RotationY":356.8299,"RotationZ":0.0},"12_GuoMao":{"OffsetX":-3.59745264,"OffsetY":0.0,"OffsetZ":-1.35211968,"RotationX":0.0,"RotationY":356.8299,"RotationZ":0.0},"13_DiWangDaSha":{"OffsetX":0.477252036,"OffsetY":0.0,"OffsetZ":3.71857429,"RotationX":0.0,"RotationY":270.184753,"RotationZ":0.0},"14_JingRongZhongXing":{"OffsetX":0.204201758,"OffsetY":0.0,"OffsetZ":-1.92322493,"RotationX":0.0,"RotationY":1.56958747,"RotationZ":0.0},"16_ShiKongChuanYue":{"OffsetX":-3.3825047,"OffsetY":0.0,"OffsetZ":1.46697176,"RotationX":0.0,"RotationY":178.482208,"RotationZ":0.0},"18_ZongBuGuangGu":{"OffsetX":-0.559406638,"OffsetY":0.0,"OffsetZ":6.13913155,"RotationX":0.0,"RotationY":269.935516,"RotationZ":0.0},"19_ShiKongChuanYue":{"OffsetX":-3.19546723,"OffsetY":0.0,"OffsetZ":1.85325563,"RotationX":0.0,"RotationY":178.2066,"RotationZ":0.0},"20_JuHe":{"OffsetX":-1.29619288,"OffsetY":0.0,"OffsetZ":6.283488,"RotationX":0.0,"RotationY":268.1551,"RotationZ":0.0},"21_HuoXing1":{"OffsetX":-8.80761,"OffsetY":0.0,"OffsetZ":5.68655729,"RotationX":0.0,"RotationY":268.1551,"RotationZ":0.0},"22_HuoXing2":{"OffsetX":-8.80761,"OffsetY":0.0,"OffsetZ":5.68655729,"RotationX":0.0,"RotationY":268.1551,"RotationZ":0.0},"23_XingChengZhongXing":{"OffsetX":-6.31454468,"OffsetY":0.0,"OffsetZ":40.8032341,"RotationX":0.0,"RotationY":175.210419,"RotationZ":0.0}}`
