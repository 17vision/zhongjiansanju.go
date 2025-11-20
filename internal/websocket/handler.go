package websocket

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
	"zjsj/internal/pkg/utils"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gfile"
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

		group.POST("/config", configHandler)

		group.GET("/rooms", getRoomsHandler)

		group.POST("/start", startHandler)

		group.POST("/scenes", scenesHandler)

		group.GET("/scenes", getScenesHandler)
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
	// var user User
	// if err = conn.ReadJSON(&user); err != nil {
	// 	g.Log().Error(ctx, "read user err:", err)
	// 	return
	// }
	var user User
	var temp = manager.getUser(string(rune(manager.lastId)))
	if temp != nil {
		user = User{
			Id:       temp.Id,
			Nickname: temp.Nickname,
			Gender:   temp.Gender,
			Avatar:   temp.Avatar,
			Extend: &UserExtend{
				IsReady: false,
			},
		}
	} else {
		r.Response.WriteStatusExit(http.StatusInternalServerError, "getUser failed")
		conn.Close()
		return
	}

	// 断开处理
	conn.SetCloseHandler(func(code int, text string) error {
		g.Log().Info(ctx, "WebSocket closed:", code, text)
		manager.removeClient(ctx, user.Id)
		return nil
	})

	// 初始化客户端信息
	client := manager.createClient(ctx, &user, conn)

	// 默认进入大厅
	room := manager.createOrJoinLobby(client)

	if client == nil {
		g.Log().Error(ctx, "CreateOrJoinLobby returned nil client")
		conn.Close()
		return
	}

	// 读写双泵
	go client.writePump(ctx)
	go client.readPump(ctx, manager)

	// 把房间里的人推送给自己
	manager.pushRoomUserList(ctx, room, user.Id)

	// 广播消息，有人进来了
	manager.broadcastUserJoined(ctx, room, &user)
}

func userHandler(r *ghttp.Request) {
	device_id := r.Get("device_id").String()

	if device_id == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传设备 id"})
		return
	}

	user := manager.getUser(device_id)

	r.Response.WriteHeader(http.StatusOK)

	r.Response.WriteJson(user)
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

func configHandler(r *ghttp.Request) {
	json := r.Get("json").String()

	_, err := gjson.DecodeToJson(json)
	if err != nil {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请传入正确的 json 格式"})
		return
	}

	// 1. 绝对路径 & 2. 确保目录存在
	path := gfile.Join(gfile.Pwd(), "storage", "posJson.json")
	if err := gfile.Mkdir(gfile.Dir(path)); err != nil {
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(map[string]any{"message": "创建目录失败"})
		return
	}

	// 3. 原子写（先写临时文件，再 rename）
	if err := gfile.PutContents(path+".tmp", json); err != nil {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "保存 json 失败"})
	}
	_ = gfile.Rename(path+".tmp", path)

	manager.posJson = json

	r.Response.WriteHeader(http.StatusOK)

	r.Response.WriteJson(map[string]any{"message": "保存 json 成功"})
}

func scenesHandler(r *ghttp.Request) {
	mapBase := r.Get("mapBase").String()
	mapScenes := r.Get("scenes").String()
	if mapBase == "" || mapScenes == "" {
		httpResponse(r, http.StatusForbidden, map[string]any{"message": "请传房间Base或Scenes"})
		return
	}

	var scenes []*Scene
	err := gjson.Unmarshal([]byte(mapScenes), &scenes)
	if err != nil {
		httpResponse(r, http.StatusForbidden, map[string]any{"message": "场景数据结构错误"})
		return
	}

	// 1. 绝对路径 & 2. 确保目录存在
	path := gfile.Join(gfile.Pwd(), "storage/scenes", mapBase+".json")
	if err := gfile.Mkdir(gfile.Dir(path)); err != nil {
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.WriteJson(map[string]any{"message": "创建目录失败"})
		return
	}

	// 3. 原子写（先写临时文件，再 rename）
	if err := gfile.PutContents(path+".tmp", mapScenes); err != nil {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "保存 json 失败"})
	}
	_ = gfile.Rename(path+".tmp", path)

	manager.scenes[mapBase] = scenes

	httpResponse(r, http.StatusOK, map[string]any{"message": "保存 json 成功"})
}

func getScenesHandler(r *ghttp.Request) {
	mapBase := r.Get("mapBase").String()
	if mapBase == "" {
		httpResponse(r, http.StatusForbidden, map[string]any{"message": "请传房间Base"})
		return
	}

	file, err := gfile.Open(gfile.Join(gfile.Pwd(), "storage/scenes/", mapBase, ".json"))
	if err != nil {
		httpResponse(r, http.StatusForbidden, map[string]any{"message": "场景不存在"})
		return
	}

	content := gfile.GetContents(file.Name())
	var scenes []*Scene
	if err = gjson.Unmarshal([]byte(content), &scenes); err != nil {
		httpResponse(r, http.StatusForbidden, map[string]any{"message": "场景获取失败,请联系管理员"})
		return
	}

	httpResponse(r, http.StatusOK, scenes)
}

type OutRoom struct {
	*Room
	UserLength    int   `json:"userLength"`
	AllReady      bool  `json:"allReady"`
	StartDuration int64 `json:"startDuration"`
}

func getRoomsHandler(r *ghttp.Request) {
	name := r.Get("name").String()
	if name == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请提供房间 id"})
	}

	// reg := regexp.MustCompile(`^[^-]+-[1-9]\d*$`)
	reg := regexp.MustCompile(fmt.Sprintf(`^%s-[1-9]\d*$`, name))

	outRooms := make([]OutRoom, 0, len(manager.rooms[RoomTypeMap]))

	for _, item1 := range manager.rooms[RoomTypeMap] {
		if reg.MatchString(item1.Id) {
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

	var scenes []byte
	mapScenes := manager.scenes[name]
	if mapScenes != nil {
		scenes, _ = gjson.Marshal(mapScenes)
	}

	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"rooms": outRooms, "scenes": string(scenes)})
}

func testHandler(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"content": "just test"})
}
