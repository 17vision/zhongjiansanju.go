package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gorilla/websocket"
)

// 定义一些数据结构
type RoomType string

const (
	RoomTypeLobby RoomType = "lobby" // 大厅
	RoomTypeMap   RoomType = "map"   // 地图
)

type Room struct {
	Id      string             `json:"id"`
	Type    RoomType           `json:"type"`
	Name    string             `json:"name"`
	Clients map[uint64]*Client `json:"clients"`
}

type Gender int

const (
	GenderUnknown Gender = 0
	GenderMale    Gender = 1
	GenderFemale  Gender = 2
)

type User struct {
	Id       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	Gender   Gender `json:"gender"`
	Avatar   string `json:"avatar"`
}

// 客户端
type Client struct {
	User     *User           `json:"user" sm:"用户信息"`
	RoomType RoomType        `json:"roomType" sm:"房间类型"`
	RoomId   string          `json:"roomId" sm:"房间Id"`
	conn     *websocket.Conn `sm:"websocket.Conn"`
	send     chan []byte     `sm:"异步写队列"`
	done     chan struct{}   `sm:"关闭信号"`
	mu       sync.Mutex      `sm:"保护 conn 置 nil"`
}

const (
	maxMessageSize = 512 * 1024
	pingPeriod     = 54 * time.Second
	writeWait      = 10 * time.Second
)

// 读泵：只负责读 + 出错时清理
func (c *Client) readPump(ctx context.Context, m *Manager) {
	defer func() {
		m.RemoveClient(ctx, c.User.Id)
	}()

	c.conn.SetReadLimit(maxMessageSize)

	_ = c.conn.SetReadDeadline(time.Now().Add(pingPeriod + 10*time.Second))

	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pingPeriod + 10*time.Second))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				g.Log("test").Async().Debugf(ctx, "readPump err: %v", err)
			}
			break
		}

		var env Envelope
		if err := gjson.Unmarshal(msg, &env); err != nil {
			g.Log("test").Async().Errorf(ctx, "消息格式错误: %v", err)
			continue
		}

		handler, ok := ReadMessageHandlers[env.Type]
		if !ok {
			g.Log("test").Async().Warningf(ctx, "消息类型不存在: %s", env.Type)
			continue
		}

		var payload interface{}
		switch env.Type {
		case "chat":
			payload = new(ChatReq)
		default:
			continue
		}

		if err := env.Data.Scan(payload); err != nil {
			g.Log("test").Async().Errorf(ctx, "消息解析失败[1] %s fail: %v", env.Type, err)
			continue
		}

		if err := handler(ctx, m, c, payload); err != nil {
			g.Log("test").Async().Errorf(ctx, "消息解析失败[2] %s fail: %v", env.Type, err)
			continue
		}
	}
}

// 写泵：消费 chan
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// chan 被关闭
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.mu.Lock()
			if c.conn == nil {
				c.mu.Unlock()
				g.Log("test").Async().Debugf(ctx, "[客户端已销毁] -> %s (room %s)", c.User.Nickname, c.RoomId)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.mu.Unlock()
				g.Log("test").Async().Debugf(ctx, "[写入数据失败] | room:%s (%s) | error: %s", c.RoomId, c.User.Nickname, err.Error())
				return
			}
			c.mu.Unlock()

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.mu.Lock()
			if c.conn == nil {
				c.mu.Unlock()
				return
			}

			// g.Log("test").Async().Debugf(ctx, "[heartbeat] Ping -> %s (room %s)", c.User.Nickname, c.RoomId)
			_ = c.conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()

		case <-c.done:
			return
		}
	}
}
