package websocket

import (
	"context"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
	"zjsj/internal/pkg/utils"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
)

var manager *Manager

func init() {
	manager = NewManager(100)
}

func BindRouters(s *ghttp.Server) {
	s.SetOpenApiPath("")
	s.SetSwaggerPath("")

	s.Group("/ws", func(group *ghttp.RouterGroup) {
		group.Middleware(service.Miiddleware().CORS)

		group.GET("/test", testHandler)

		group.GET("/room/users", getRoomUsersHandler)

		group.POST("/start", startHandler)

		group.POST("/stop", stopHandler)

		// group.POST("/config", configHandler)

		// group.POST("/scenes", storeScenesHandler)

		// group.GET("/scenes", getScenesHandler)
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
		r.Response.WriteStatusExit(http.StatusInternalServerError, "getUser failed")
		conn.Close()
		return
	}

	// 断开处理
	conn.SetCloseHandler(func(code int, text string) error {
		g.Log().Info(ctx, "SetCloseHandler WebSocket closed:", code, text)
		manager.removeClient(ctx, user.Id)
		return nil
	})

	g.Log("test").Async().Infof(ctx, "%s【%d】已连接", user.Nickname, user.Id)

	user.Extend = &UserExtend{IsStart: false, ConnectTime: time.Now().Unix(), ConnectIp: clientIPv4(r)}

	// 初始化客户端信息
	client := manager.createClient(ctx, &user, conn)

	if client == nil {
		g.Log().Error(ctx, "CreateOrJoinLobby returned nil client")
		conn.Close()
		return
	}

	// 读写双泵
	go client.writePump(ctx)
	go client.readPump(ctx, manager)

	manager.joinRoom(ctx)
}

func httpResponse(r *ghttp.Request, status int, data interface{}) {
	r.Response.WriteHeader(status)
	r.Response.WriteJson(data)
}

func testHandler(r *ghttp.Request) {
	httpResponse(r, http.StatusOK, map[string]any{"content": "just test"})
}

func startHandler(r *ghttp.Request) {
	userIds := r.Get("userIds").String()

	if userIds == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传用户 id,多个用户 id 用 , 连接"})
		return
	}

	uids := strings.Split(userIds, ",")

	result := manager.start(r.GetCtx(), uids)

	r.Response.WriteHeader(http.StatusOK)

	r.Response.WriteJson(map[string]any{"result": result})

	// 统计
	startRecord(r.GetCtx(), uids)
}

func startRecord(ctx context.Context, uids []string) {
	var clients []*Client
	for _, item := range uids {
		userId := gconv.Uint64(item)

		client, ok := manager.room.Clients[userId]
		if ok && client != nil && client.User.Extend.IsStart {
			clients = append(clients, client)
		}
	}

	if len(clients) == 0 {
		return
	}

	for _, item := range clients {
		var sn string
		if item.User.Sn != "" {
			sn = item.User.Sn
		} else {
			sn = gconv.String(item.User.Id)
		}

		data := g.Map{
			"name":       "hxlc3",
			"ip":         item.User.Extend.ConnectIp,
			"connect_at": gtime.NewFromTimeStamp(item.User.Extend.ConnectTime).Format("Y-m-d H:i:s"),
			"start_at":   gtime.NewFromTimeStamp(item.User.Extend.StartTime).Format("Y-m-d H:i:s"),
			"sn":         sn,
		}
		r, err := g.Client().Post(ctx, "https://game.17vision.com/api/game/start_records", data)

		if err != nil {
			continue
		}
		defer r.Close()

		result := r.ReadAllString()
		type GameStartRecord struct {
			ID        uint64 `json:"id"`
			Name      string `json:"name"`
			ConnectAt string `json:"connect_at"`
			StartAt   string `json:"start_at"`
		}

		var gameStartRecord GameStartRecord
		err = gjson.Unmarshal([]byte(result), &gameStartRecord)
		if err == nil {
			item.User.Extend.RecordId = gameStartRecord.ID

			g.Log("test").Async().Infof(ctx, "%s 开始游戏已统计。统计 id 是 %d", item.User.Nickname, gameStartRecord.ID)
		} else {
			g.Log("test").Async().Errorf(ctx, "%s 开始游戏统计失败。错误是 %s", item.User.Nickname, err.Error())
		}
	}
}

func stopHandler(r *ghttp.Request) {
	userIds := r.Get("userIds").String()

	if userIds == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传用户 id,多个用户 id 用 , 连接"})
		return
	}

	uids := strings.Split(userIds, ",")

	result := manager.stop(r.GetCtx(), uids)

	r.Response.WriteHeader(http.StatusOK)

	r.Response.WriteJson(map[string]any{"result": result})

	// 统计
	// startRecord(r.GetCtx(), uids)
}

type OutUser struct {
	*User
	StartDuration   int64 `json:"startDuration"`
	ConnectDuration int64 `json:"connectDuration"`
}

func getRoomUsersHandler(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusOK)

	users := make([]*OutUser, 0, len(manager.clients))
	now := time.Now().Unix()

	for _, c := range manager.clients {

		var duration int64
		if c.User.Extend.StartTime > 0 {
			duration = now - c.User.Extend.StartTime
		}

		users = append(users, &OutUser{
			User:            c.User,
			StartDuration:   duration,
			ConnectDuration: now - c.User.Extend.ConnectTime,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		// 第一优先级：IsStart 降序（true > false）
		if users[i].Extend.IsStart != users[j].Extend.IsStart {
			return users[i].Extend.IsStart
		}

		// 第二优先级：ConnectDuration 降序
		if users[i].ConnectDuration != users[j].ConnectDuration {
			return users[i].ConnectDuration > users[j].ConnectDuration
		}

		// 第三优先级：Id 升序
		return int64(users[i].Id) < int64(users[j].Id)
	})

	r.Response.WriteJson(map[string]any{"users": users, "waiters": manager.waitUsers})
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

// func configHandler(r *ghttp.Request) {
// 	json := r.Get("json").String()

// 	_, err := gjson.DecodeToJson(json)
// 	if err != nil {
// 		r.Response.WriteHeader(http.StatusForbidden)
// 		r.Response.WriteJson(map[string]any{"message": "请传入正确的 json 格式"})
// 		return
// 	}

// 	// 1. 绝对路径 & 2. 确保目录存在
// 	path := gfile.Join(gfile.Pwd(), "storage", "posJson.json")
// 	if err := gfile.Mkdir(gfile.Dir(path)); err != nil {
// 		r.Response.WriteHeader(http.StatusInternalServerError)
// 		r.Response.WriteJson(map[string]any{"message": "创建目录失败"})
// 		return
// 	}

// 	// 3. 原子写（先写临时文件，再 rename）
// 	if err := gfile.PutContents(path+".tmp", json); err != nil {
// 		r.Response.WriteHeader(http.StatusForbidden)
// 		r.Response.WriteJson(map[string]any{"message": "保存 json 失败"})
// 	}
// 	_ = gfile.Rename(path+".tmp", path)

// 	// manager.posJson = json

// 	r.Response.WriteHeader(http.StatusOK)

// 	r.Response.WriteJson(map[string]any{"message": "保存 json 成功"})
// }

// func storeScenesHandler(r *ghttp.Request) {
// 	mapBase := r.Get("mapBase").String()
// 	mapScenes := r.Get("scenes").String()
// 	if mapBase == "" || mapScenes == "" {
// 		httpResponse(r, http.StatusForbidden, map[string]any{"message": "请传房间Base或Scenes"})
// 		return
// 	}

// 	var scenes []*Scene
// 	err := gjson.Unmarshal([]byte(mapScenes), &scenes)
// 	if err != nil {
// 		httpResponse(r, http.StatusForbidden, map[string]any{"message": "场景数据结构错误"})
// 		return
// 	}

// 	// 1. 绝对路径 & 2. 确保目录存在
// 	path := gfile.Join(gfile.Pwd(), "storage/scenes", mapBase+".json")
// 	if err := gfile.Mkdir(gfile.Dir(path)); err != nil {
// 		r.Response.WriteHeader(http.StatusInternalServerError)
// 		r.Response.WriteJson(map[string]any{"message": "创建目录失败"})
// 		return
// 	}

// 	// 3. 原子写（先写临时文件，再 rename）
// 	if err := gfile.PutContents(path+".tmp", mapScenes); err != nil {
// 		r.Response.WriteHeader(http.StatusForbidden)
// 		r.Response.WriteJson(map[string]any{"message": "保存 json 失败"})
// 	}
// 	_ = gfile.Rename(path+".tmp", path)

// 	// manager.scenes[mapBase] = scenes

// 	httpResponse(r, http.StatusOK, map[string]any{"message": "保存 json 成功"})
// }

// func getScenesHandler(r *ghttp.Request) {
// 	// mapBase := r.Get("mapBase").String()
// 	// if mapBase == "" {
// 	// 	httpResponse(r, http.StatusForbidden, map[string]any{"message": "请传房间Base"})
// 	// 	return
// 	// }

// 	// for key, scenes := range manager.scenes {
// 	// 	if key == mapBase {
// 	// 		httpResponse(r, http.StatusOK, scenes)
// 	// 		return
// 	// 	}
// 	// }

// 	// // 这个是从 file 里读
// 	// fileName := gfile.Join(gfile.Pwd(), "storage/scenes/", mapBase+".json")

// 	// file, err := gfile.Open(gfile.Join(fileName))
// 	// if err != nil {
// 	// 	httpResponse(r, http.StatusForbidden, map[string]any{"message": "场景不存在", "error": err.Error()})
// 	// 	return
// 	// }

// 	// content := gfile.GetContents(file.Name())
// 	// var scenes []*Scene
// 	// if err = gjson.Unmarshal([]byte(content), &scenes); err != nil {
// 	// 	httpResponse(r, http.StatusForbidden, map[string]any{"message": "场景获取失败,请联系管理员"})
// 	// 	return
// 	// }
// 	// httpResponse(r, http.StatusOK, scenes)
// }

// func getRoomsHandler(r *ghttp.Request) {
// 	// mapBase := r.Get("mapBase").String()
// 	// if mapBase == "" {
// 	// 	r.Response.WriteHeader(http.StatusForbidden)
// 	// 	r.Response.WriteJson(map[string]any{"message": "请传房间Base"})
// 	// }

// 	// // reg := regexp.MustCompile(`^[^-]+-[1-9]\d*$`)
// 	// // reg := regexp.MustCompile(fmt.Sprintf(`^%s-[1-9]\d*$`, name))

// 	// outRooms := make([]OutRoom, 0, len(manager.rooms[RoomTypeMap]))

// 	// for _, item1 := range manager.rooms[RoomTypeMap] {
// 	// 	if item1.MapBase == mapBase {
// 	// 		readyNum := 0
// 	// 		allNum := 0
// 	// 		for _, item2 := range item1.Clients {
// 	// 			allNum++
// 	// 			if item2.User.Extend.IsReady {
// 	// 				readyNum++
// 	// 			}
// 	// 		}

// 	// 		var startDuration int64 = 0
// 	// 		if item1.StartTime > 0 {
// 	// 			startDuration = time.Now().Unix() - item1.StartTime
// 	// 		}

// 	// 		outRooms = append(outRooms, OutRoom{
// 	// 			Room:          item1,
// 	// 			UserLength:    allNum,
// 	// 			AllReady:      readyNum == allNum,
// 	// 			StartDuration: startDuration,
// 	// 		})
// 	// 	}
// 	// }

// 	// var scenes []byte
// 	// mapScenes := manager.scenes[mapBase]
// 	// if mapScenes != nil {
// 	// 	scenes, _ = gjson.Marshal(mapScenes)
// 	// }

// 	// r.Response.WriteHeader(http.StatusOK)
// 	// r.Response.WriteJson(map[string]any{"rooms": outRooms, "scenes": string(scenes)})
// }
