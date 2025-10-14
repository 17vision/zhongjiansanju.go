package websocket

import (
	"net/http"
	"zjsj/internal/pkg/utils"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
)

var manager *Manager

func init() {
	manager = NewManager(100, 15)
}

func BindRouters(s *ghttp.Server) {
	s.SetOpenApiPath("")
	s.SetSwaggerPath("")

	s.Group("/ws", func(group *ghttp.RouterGroup) {
		group.GET("/test", testHandler)
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

	// 断开处理
	conn.SetCloseHandler(func(code int, text string) error {
		g.Log().Info(ctx, "WebSocket closed:", code, text)
		manager.RemoveClient(ctx, user.Id)
		return nil
	})

	var room *Room

	room, client := manager.CreateOrJoinLobby(&user, conn)

	if client == nil {
		g.Log().Error(ctx, "CreateOrJoinLobby returned nil client")
		conn.Close()
		return
	}

	// 读写双泵
	go client.writePump(ctx)
	go client.readPump(ctx, manager)

	// 把房间里的人推送给自己
	manager.PushRoomUserList(ctx, room, user.Id)

	// 广播消息，有人进来了
	manager.BroadcastUserJoined(ctx, room, &user)
}

func testHandler(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"content": "just test"})
}
