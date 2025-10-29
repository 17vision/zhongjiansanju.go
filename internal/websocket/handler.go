package websocket

import (
	"net/http"
	"regexp"
	"time"
	"zjsj/internal/pkg/utils"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
)

var manager *Manager

func init() {
	manager = NewManager(50, 6)
}

func BindRouters(s *ghttp.Server) {
	s.SetOpenApiPath("")
	s.SetSwaggerPath("")

	s.Group("/ws", func(group *ghttp.RouterGroup) {
		group.Middleware(service.Miiddleware().CORS)

		group.GET("/test", testHandler)

		group.POST("/user", userHandler)

		group.POST("/start", startHandler)

		group.GET("/rooms", getRoomsHandler)
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

type OutRoom struct {
	*Room
	UserLength    int   `json:"userLength"`
	AllReady      bool  `json:"allReady"`
	StartDuration int64 `json:'startDuration'`
}

func getRoomsHandler(r *ghttp.Request) {
	name := r.Get("name").String()
	if name == "" {
		r.Response.WriteHeader(http.StatusForbidden)
		r.Response.WriteJson(map[string]any{"message": "请提供房间 id"})
	}

	reg := regexp.MustCompile(`^[^-]+-[1-9]\d*$`)

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

			outRooms = append(outRooms, OutRoom{
				Room:          item1,
				UserLength:    allNum,
				AllReady:      readyNum == allNum,
				StartDuration: time.Now().Unix() - item1.StartTime,
			})
		}
	}

	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"rooms": outRooms})
}

func testHandler(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"content": "just test"})
}
