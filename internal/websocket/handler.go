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
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gorilla/websocket"
)

var manager *Manager
var unityService *UnityService

func init() {
	manager = NewManager(50, 8)
	unityService = NewUnityService()
	// 云服务器创建结果（轮询/回调）通知房间
	unityService.SetHooks(manager.onAllocationReady, manager.onAllocationFailed)
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

		// 云服务器分配：创建/查询/释放（与 start/stop 并列）
		group.POST("/allocation", createAllocationHandler)

		group.GET("/allocation", getAllocationHandler)

		group.POST("/allocation/delete", deleteAllocationHandler)

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

	// 与轮询共用幂等完成逻辑：allocated 补全 ip，failed 记录原因
	unityService.HandleCallback(allocUUID, ip, ports, status, req.Msg)

	r.Response.WriteJson(g.Map{
		"code": 0,
		"msg":  "ok",
	})
}

// createAllocationHandler HTTP 创建云服务器：
// 传 roomId 则给已有房间补建服务器；不传则新建房间并触发创建。
// 立即返回分配信息，ip 为空表示创建中，可轮询 GET /ws/allocation 或等回调。
func createAllocationHandler(r *ghttp.Request) {
	ctx := r.GetCtx()
	roomId := r.Get("roomId").String()
	mapBase := r.Get("mapBase").String()
	if mapBase == "" {
		mapBase = defaultMapBase
	}

	var room *Room
	if roomId != "" {
		manager.mu.RLock()
		for _, item := range manager.rooms[RoomTypeMap] {
			if item.Id == roomId {
				room = item
				break
			}
		}
		manager.mu.RUnlock()

		if room == nil {
			r.Response.WriteHeader(http.StatusForbidden)
			r.Response.WriteJson(map[string]any{"message": "房间不存在"})
			return
		}
		manager.ensureRoomAllocation(ctx, room)
	} else {
		room = manager.CreateRoomWithServer(ctx, mapBase)
	}

	alloc := unityService.GetRoomAllocation(room.Id)

	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{
		"roomId":     room.Id,
		"allocation": alloc, // ip 为空表示创建中
	})
}

// getAllocationHandler 查询分配状态：传 uuid 查单个，不传查全部
func getAllocationHandler(r *ghttp.Request) {
	id := r.Get("uuid").String()

	r.Response.WriteHeader(http.StatusOK)
	if id != "" {
		r.Response.WriteJson(map[string]any{"allocation": unityService.GetAllocation(id)})
		return
	}
	r.Response.WriteJson(map[string]any{"allocations": unityService.ListAllocations()})
}

// deleteAllocationHandler 释放云服务器
func deleteAllocationHandler(r *ghttp.Request) {
	id := r.Get("uuid").String()
	if id == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传 uuid"})
		return
	}

	manager.releaseRoomServer(r.GetCtx(), id)

	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"result": true})
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

	// 普通用户：先取设备使用信息（昵称/模型），决定是否直接进特殊房间
	if user.Type == TypeUser {
		// 干掉设置昵称 模型的功能
		// res, err := service.GlassesUse().One(ctx, user.EquipmentSn)
		// if err != nil {
		// 	g.Log().Error(ctx, "初始化用户信息失败", err.Error())
		// 	conn.Close()
		// 	return
		// }

		// if res == nil {
		// 	g.Log().Error(ctx, "初始化用户信息失败", "设备和设备关系不存在")
		// 	conn.Close()
		// 	return
		// }

		// g.Log().Info(ctx, "相关设备信息：", gjson.MustEncodeString(res))

		// user.Nickname = res.GlassesUse.Nickname
		// user.Model = res.GlassesUse.Model
		// user.GlassesUseId = res.GlassesUse.Id

		// if err = service.GlassesUse().UpdateStatus(ctx, int64(res.GlassesUse.Id), 2); err != nil {
		// 	g.Log().Error(ctx, "更新设备状态为 2 失败", err.Error())
		// 	conn.Close()
		// 	return
		// }
		// g.Log().Info(ctx, "更新设备状态为 2 成功")

		user.Nickname = "ClientID"
		user.Model = "Player"
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

	// 读写双泵
	go client.writePump(ctx)
	go client.readPump(ctx, manager)

	if user.Type == TypeUser {
		var room *Room
		if user.Nickname != "" && user.Model != "" {
			// 配置了昵称和模型：直接进特殊房间，没有可用房间就新建并异步创建云服务器
			room = manager.createOrJoinMap(ctx, client, defaultMapBase)
			user.Allocation = unityService.GetRoomAllocation(room.Id)
		} else {
			// 默认进入大厅
			room = manager.createOrJoinLobby(client)
		}

		// 发给自己，加入了房间（ip 为空表示云服务器还在创建中）
		manager.mu.RLock()
		var port int
		if room.GameServerClient != nil {
			port = room.GameServerClient.User.Port
		}
		manager.mu.RUnlock()
		manager.unicastAsync(ctx, client, WSMessage{
			Type: MsgTypeJoined,
			Data: UserEventData{RoomId: room.Id, Ip: room.Ip, Port: port},
		})

		// 把房间里的人推送给自己
		manager.pushRoomUserList(ctx, room, user)

		// 广播消息，有人进来了
		manager.broadcastUserJoined(ctx, room, &user)

		if room.Type == RoomTypeMap {
			// 推送云服务器状态：创建中/就绪/失败
			manager.pushServerStatus(ctx, room, client)
		} else {
			// 大厅用户可以进房间
			manager.pushUserCanEnterRoom(ctx, client)
		}
	} else {
		// 游戏服务器回连：按分配 uuid 绑定归属房间
		manager.addGameClient(client)
		manager.bindGameServer(ctx, client)

		// 服务器进来，给大厅用户发送可进房间命令
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
	UserLength    int         `json:"userLength"`
	AllReady      bool        `json:"allReady"`
	StartDuration int64       `json:"startDuration"`
	ServerReady   bool        `json:"serverReady"`  // 云服务器就绪（有 ip 且游戏服务器已回连）才能开始游戏
	Allocation    *Allocation `json:"allocation"`   // 云服务器分配信息（ip 为空表示创建中）
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
				ServerReady:   item1.Ip != "" && item1.GameServerClient != nil,
				Allocation:    unityService.GetRoomAllocation(item1.Id),
			})
		}
	}

	type GameServerClient struct {
		Id        uint64 `json:"id"`
		Port      int    `json:"port"`
		Ip        string `json:"ip"`
		IsServer  bool   `json:"isServer"`
		RoomId    string `json:"roomId"`
		CreatedAt string `json:"createdAt"`
	}

	outGameServerClient := make([]GameServerClient, 0, len(manager.gameServerClients))
	for _, item := range manager.gameServerClients {
		// 用服务器自己的分配 ip，其次用 WS 连接来源 IP；不用全局缓存兜底，避免多服务器时显示串 IP
		ip := ""
		if item.User.Allocation != nil && item.User.Allocation.Ip != "" {
			ip = item.User.Allocation.Ip
		} else if item.User.Extend != nil {
			ip = item.User.Extend.ConnectIp
		}
		outGameServerClient = append(outGameServerClient, GameServerClient{
			Id:        item.User.Id,
			Port:      item.User.Port,
			Ip:        ip,
			IsServer:  item.User.Extend.IsServer,
			RoomId:    item.RoomId,
			CreatedAt: gtime.New(item.CreatedAt).Format("Y-m-d H:i:s"),
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
