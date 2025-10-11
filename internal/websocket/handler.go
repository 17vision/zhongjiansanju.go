package websocket

import (
	"net/http"
	"time"
	"zjsj/internal/pkg/utils"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
)

var DefaultManager *Manager

func init() {
	DefaultManager = NewManager(100, 15)
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
		DefaultManager.RemoveClient(ctx, user.Id)
		return nil
	})

	var room *Room

	// 兜底
	defer func() {
		DefaultManager.RemoveClient(ctx, user.Id)
	}()

	room, _ = DefaultManager.CreateOrJoinLobby(&user)

	var client *Client
	if c0, ok := room.Clients[user.Id]; ok {
		c0.Conn = conn
		c0.LastHeartbeat = time.Now().UnixMilli()
		client = c0
	}

	// 通知自己，加入了房间
	DefaultManager.SendMessage(ctx, conn, WSMessage{
		Type: MsgTypeJoined,
		Data: JoinedData{RoomId: room.Id},
	})

	// 把房间里的人推送给自己
	DefaultManager.PushRoomUserListTo(ctx, room, user.Id)

	// 广播消息，有人进来了
	DefaultManager.BroadcastUserJoinedExcept(ctx, room, &user)

	// 启动心跳检测
	go HeartbeatChecker(client, func() {
		conn.Close()
	})

	// 断线重连控制
	if client != nil {
		client.ReconnectCount++
		if client.ReconnectCount > MaxReconnect {
			// websocket.StatusPolicyViolation, "reconnect too many times"
			conn.Close()
			return
		}
	}

	// 消息接收
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			g.Log().Info(ctx, "Connection closed or error occurred:", err)
			break
		}

		if msg["type"] == MsgTypePing {
			if client != nil {
				client.LastHeartbeat = time.Now().UnixMilli()

				DefaultManager.SendMessage(ctx, conn, WSMessage{
					Type: MsgTypePong,
					Data: nil,
				})
			}
			continue
		}

		switch msg["type"] {
		case "goto_map":
			// 先广播离开原房间
			if room != nil {
				DefaultManager.BroadcastUserLeft(ctx, room, &user)
			}

			base := msg["map"].(string)
			room, _ = DefaultManager.CreateOrJoinMap(&user, base)
			DefaultManager.JoinRoom(user.Id, room.Id)
			if client, ok := room.Clients[user.Id]; ok {
				client.Conn = conn
			}

			DefaultManager.SendMessage(ctx, conn, WSMessage{
				Type: MsgTypeMoved,
				Data: MovedData{RoomID: room.Id},
			})
			DefaultManager.PushRoomUserListTo(ctx, room, user.Id)
			DefaultManager.BroadcastUserJoinedExcept(ctx, room, &user)
		}
	}
}

func testHandler(r *ghttp.Request) {
	r.Response.WriteHeader(http.StatusOK)
	r.Response.WriteJson(map[string]any{"content": "just test"})
}
