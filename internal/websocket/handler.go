package websocket

import (
	"encoding/json"
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
var unityService *UnityService

func init() {
	manager = NewManager(50, 8)
	unityService = NewUnityService()
}

type AllocationCallbackReq struct {
	// 外层主体（真正要用的）
	Uuid            string   `json:"uuid"`
	Ip              string   `json:"ip"`
	GameServerPorts []string `json:"gameServerPorts"`
	Status          string   `json:"status"` // allocated / failed
	Msg             string   `json:"msg"`
	UosAppId        string   `json:"uosAppId"`
	ProfileId       string   `json:"profileId"`
	RegionId        string   `json:"regionId"`

	// 报文里附带的 description，可选加，平台好像只是冗余携带
	Description struct {
		Uuid            string   `json:"uuid"`
		Ip              string   `json:"ip"`
		GameServerPorts []string `json:"gameServerPorts"`
		Status          string   `json:"status"`
		Msg             string   `json:"msg"`
		UosAppId        string   `json:"uosAppId"`
		ProfileId       string   `json:"profileId"`
		RegionId        string   `json:"regionId"`
	} `json:"description"`
}

func BindRouters(s *ghttp.Server) {
	s.SetOpenApiPath("")
	s.SetSwaggerPath("")

	s.Group("/ws", func(group *ghttp.RouterGroup) {
		group.Middleware(service.Middleware().CORS)

		group.GET("/test", testHandler)

		group.POST("/user", userHandler)

		group.GET("/rooms", getRoomsHandler)

		group.POST("/start", startHandler)

		group.POST("/stop", stopHandler)

		group.POST("/allocation/callback", allocationCallback)
	})

	s.BindHandler("/ws/connect", websocketHandler)
}

func allocationCallback(r *ghttp.Request) {
	var req AllocationCallbackReq
	body := r.GetBody()
	if err := json.Unmarshal(body, &req); err != nil {
		g.Log().Errorf(r.GetCtx(), "parse json failed, err:%v, raw:%s", err, string(body))
		r.Response.WriteJson(g.Map{
			"code": 400,
			"msg":  "parse json failed: " + err.Error(),
		})
		return
	}

	// 业务逻辑不变
	allocUUID := req.Uuid
	status := req.Status
	ip := req.Ip
	ports := req.GameServerPorts

	g.Log().Infof(r.GetCtx(), "alloc callback, uuid=%s, status=%s, ip=%s, ports=%+v", allocUUID, status, ip, ports)

	switch status {
	case "allocated":
		// 更新分配成功
	case "failed":
		// 更新失败，记录 req.Msg
	default:
		g.Log().Warningf(r.GetCtx(), "unknown allocation status: %s", status)
	}

	r.Response.WriteJson(g.Map{
		"code": 0,
		"msg":  "ok",
	})
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
		g.Log().Error(ctx, "Read user err:", err)
		r.Response.WriteStatusExit(http.StatusInternalServerError, "用户数据错误"+err.Error())
		return
	}

	// 普通客户端用户激活
	if user.Type == TypeUser {
		if unityService.GetStatus() {
			g.Log().Error(ctx, "服务器分配中，请稍后再连")
			r.Response.WriteStatusExit(http.StatusInternalServerError, "服务器分配中，请稍后再连")
			return
		}

		allocation, err := unityService.GetAllocation(ctx)
		if err != nil {
			g.Log().Error(ctx, "分配服务器失败[1]:", err)
			r.Response.WriteStatusExit(http.StatusInternalServerError, "分配服务器失败:"+err.Error())
			return
		}

		if allocation == nil {
			g.Log().Error(ctx, "分配服务器失败[2]:", err)
			r.Response.WriteStatusExit(http.StatusInternalServerError, "分配服务器失败")
			return
		}

		if len(manager.gameServerClients) == 0 {
			g.Log().Error(ctx, "等待云服务器连接")
			conn.Close()
			r.Response.WriteStatusExit(http.StatusInternalServerError, "等待云服务器连接")
			return
		}
		user.Allocation = allocation
	} else {
		allocation, err := unityService.GetAllocation(ctx)
		if err != nil {
			g.Log().Error(ctx, "[2]分配服务器失败[1]:", err)
			r.Response.WriteStatusExit(http.StatusInternalServerError, "分配服务器失败:"+err.Error())
			return
		}

		if allocation == nil {
			g.Log().Error(ctx, "[2]分配服务器失败[2]:", err)
			r.Response.WriteStatusExit(http.StatusInternalServerError, "分配服务器失败")
			return
		}
		user.Allocation = allocation
	}

	user.Extend = &UserExtend{
		IsReady:     false,
		IsServer:    false,
		ConnectTime: time.Now().Unix(),
		ConnectIp:   clientIPv4(r),
	}

	g.Log().Info(ctx, "用户进入：", gjson.MustEncodeString(user))

	if user.Type == TypeUser {
		res, err := service.GlassesUse().One(ctx, user.EquipmentSn)

		if err != nil {
			g.Log().Error(ctx, "初始化用户信息失败", err.Error())
			return
		}

		if res == nil {
			g.Log().Error(ctx, "初始化用户信息失败", "设备和设备关系不存在")
			return
		}

		g.Log().Info(ctx, "相关设备信息：", gjson.MustEncodeString(res))

		user.Nickname = res.GlassesUse.Nickname
		user.Model = res.GlassesUse.Model
		user.GlassesUseId = res.GlassesUse.Id

		err = service.GlassesUse().UpdateStatus(ctx, int64(res.GlassesUse.Id), 2)
		if err != nil {
			g.Log().Error(ctx, "更新设备状态为 2 失败", err.Error())
			return
		} else {
			g.Log().Info(ctx, "更新设备状态为 2 成功")
		}
		// user.Nickname = GetName()
	}

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

		// 游戏服务器客户端连上，用其真实 IP 动态更新（变化才更新并写缓存）
		manager.updateGameServerIp(ctx, user.Allocation.Ip)
	}

	// 读写双泵
	go client.writePump(ctx)
	go client.readPump(ctx, manager)

	if room != nil {
		// 把房间里的人推送给自己
		manager.pushRoomUserList(ctx, room, user)

		// 广播消息，有人进来了
		manager.broadcastUserJoined(ctx, room, &user)

		// 如果用户可以进房间就发给用户
		manager.pushUserCanEnterRoom(ctx, client)
	} else {
		manager.unicastAsync(ctx, client, WSMessage{
			Type: MsgTypeUserJoined,
			Data: UserEventData{UserId: user.Id, Ip: manager.gameServerIp()},
		})

		// 服务器进来，给未进入房间的用户发送命令
		manager.checkUserCanEnterRoom(ctx)
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
			Ip:       manager.gameServerIp(),
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
