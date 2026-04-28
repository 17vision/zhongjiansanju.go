package websocket

import (
	"net"
	"net/http"
	"time"
	"zjsj/internal/pkg/utils"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
)

var manager *Manager

func init() {
	manager = NewManager(50, 8)
}

func BindRouters(s *ghttp.Server) {
	s.SetOpenApiPath("")
	s.SetSwaggerPath("")

	s.Group("/ws", func(group *ghttp.RouterGroup) {
		group.Middleware(service.Miiddleware().CORS)

		group.GET("/test", testHandler)

		group.POST("/user", userHandler)

		group.GET("/rooms", getRoomsHandler)

		group.POST("/start", startHandler)

		group.POST("/stop", stopHandler)
	})

	s.BindHandler("/ws/connect", websocketHandler)
}

func websocketHandler(r *ghttp.Request) {
	ctx := r.GetCtx()

	wsUpGrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			queryParams := r.URL.Query()
			time := queryParams.Get("time")
			sign := queryParams.Get("sign")
			return utils.Check(ctx, time, sign)
		},
		// Error handler for upgrade failures
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			g.Log().Error(ctx, "WebSocket upgrade error: ", reason)
		},
	}

	conn, err := wsUpGrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		g.Log().Error(ctx, "WebSocket upgrade failed: ", err)
		r.Response.WriteStatusExit(http.StatusInternalServerError, "WebSocket upgrade failed")
		return
	}

	// 收到的第一个消息必须是用户信息
	var user User
	if err = conn.ReadJSON(&user); err != nil {
		g.Log().Error(ctx, "read user err:", err)
		return
	}

	user.Extend = &UserExtend{
		IsReady:     false,
		IsServer:    false,
		ConnectTime: time.Now().Unix(),
		ConnectIp:   clientIPv4(r),
	}

	g.Log().Info(ctx, "用户进入：", gjson.MustEncodeString(user))

	// 断开处理
	conn.SetCloseHandler(func(code int, text string) error {
		g.Log().Info(ctx, "WebSocket closed:", code, text)
		manager.removeClient(ctx, user.Id)
		return nil
	})

	// 初始化客户端信息
	client := manager.createClient(ctx, &user, conn)

	if client == nil {
		g.Log().Error(ctx, "returned nil client")
		conn.Close()
		return
	}

	// 默认进入大厅
	var room *Room
	if user.Type == TypeUser {
		room = manager.createOrJoinLobby(client)
	} else {
		manager.addGameClient(client)
	}

	// 读写双泵
	go client.writePump(ctx)
	go client.readPump(ctx, manager)

	if room != nil {
		// 把房间里的人推送给自己
		manager.pushRoomUserList(ctx, room, user.Id)

		// 广播消息，有人进来了
		manager.broadcastUserJoined(ctx, room, &user)
	} else {
		manager.unicastAsync(ctx, client, WSMessage{
			Type: MsgTypeUserJoined,
			Data: UserEventData{UserId: user.Id, Ip: manager.gameConfig.Ip},
		})
	}
}

func userHandler(r *ghttp.Request) {
	device_id := r.Get("device_id").String()

	if device_id == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传设备 id"})
		return
	}

	// user := manager.getUser(device_id)

	// r.Response.WriteHeader(http.StatusOK)

	// r.Response.WriteJson(user)
}

func httpResponse(r *ghttp.Request, status int, data interface{}) {
	r.Response.WriteHeader(status)
	r.Response.WriteJson(data)
}

func startHandler(r *ghttp.Request) {
	roomId := r.Get("roomId").String()

	if roomId == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传房间 id"})
		return
	}

	result := manager.start(r.GetCtx(), roomId)

	r.Response.WriteHeader(http.StatusOK)

	r.Response.WriteJson(map[string]any{"result": result})
}

func stopHandler(r *ghttp.Request) {
	roomId := r.Get("roomId").String()

	if roomId == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传房间 id"})
		return
	}

	result := manager.stop(r.GetCtx(), roomId)

	r.Response.WriteHeader(http.StatusOK)

	r.Response.WriteJson(map[string]any{"result": result})
}

type OutRoom struct {
	*Room
	UserLength    int   `json:"userLength"`
	AllReady      bool  `json:"allReady"`
	StartDuration int64 `json:"startDuration"`
}

func getRoomsHandler(r *ghttp.Request) {
	mapBase := r.Get("mapBase").String()
	if mapBase == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传房间Base"})
	}

	// reg := regexp.MustCompile(`^[^-]+-[1-9]\d*$`)
	// reg := regexp.MustCompile(fmt.Sprintf(`^%s-[1-9]\d*$`, name))

	outRooms := make([]OutRoom, 0, len(manager.rooms[RoomTypeMap]))

	for _, item1 := range manager.rooms[RoomTypeMap] {
		if item1.MapBase == mapBase {
			readyNum := 0
			allNum := 0
			for _, item2 := range item1.Clients {
				allNum++
				if item2.User.Extend.IsReady {
					readyNum++
				}
			}

			var startDuration int64 = 0
			if item1.StartTime > 0 {
				startDuration = time.Now().Unix() - item1.StartTime
			}

			outRooms = append(outRooms, OutRoom{
				Room:          item1,
				UserLength:    allNum,
				AllReady:      readyNum == allNum,
				StartDuration: startDuration,
			})
		}
	}

	type GameServerClient struct {
		Id       uint64 `json:"id"`
		Port     int    `json:"port"`
		Ip       string `json:"ip"`
		IsServer bool   `json:"isServer"`
		RoomId   string `json:"roomId"`
	}

	outGameServerClient := make([]GameServerClient, 0, len(manager.gameServerClients))
	for _, item := range manager.gameServerClients {
		outGameServerClient = append(outGameServerClient, GameServerClient{
			Id:       item.User.Id,
			Port:     item.User.Port,
			Ip:       manager.gameConfig.Ip,
			IsServer: item.User.Extend.IsServer,
			RoomId:   item.RoomId,
		})
	}

	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{
		"rooms":   outRooms,
		"servers": outGameServerClient,
	})
}

func clientIPv4(r *ghttp.Request) string {
	ip := r.GetClientIp() // 已经处理过 X-Forwarded-For

	// 去掉端口（极少情况 RemoteAddr 会带端口）
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	// 解析地址
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "" // 非法地址
	}

	// ::1 -> 127.0.0.1
	if parsed.IsLoopback() {
		return "127.0.0.1"
	}

	// ::ffff:192.168.1.100 -> 192.168.1.100
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}

	// 纯 IPv6 无法映射，按需返回空串或原地址
	return "" // 或者 return ip，看你业务
}

func testHandler(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"content": "just test"})
}
