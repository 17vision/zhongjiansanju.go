package websocket

import (
	"context"
	"sync"
	"sync/atomic"
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

type RoomStatus int

const (
	RoomStatusWating  RoomStatus = 0
	RoomStatusPlaying RoomStatus = 1
	RoomStatusStopped RoomStatus = 2
)

type Room struct {
	Id               string             `json:"id"`
	MapBase          string             `json:"mapBase"`
	Type             RoomType           `json:"type"`
	Name             string             `json:"name"`
	Capacity         int                `json:"capacity"`
	Clients          map[uint64]*Client `json:"clients"`
	GameServerClient *Client            `json:"gameServerClient"`
	Status           RoomStatus         `json:"roomStatus"`
	StartTime        int64              `json:"startTime"`
}

type Gender int

const (
	GenderUnknown Gender = 0
	GenderMale    Gender = 1
	GenderFemale  Gender = 2
)

// 用户类型
type UserType int

const (
	TypeUser       UserType = 1
	TypeGameServer UserType = 2
)

// 用户信息
type User struct {
	Type     UserType    `json:"type"`
	Port     int         `json:"port"`
	Id       uint64      `json:"id"`
	Nickname string      `json:"nickname"`
	Gender   Gender      `json:"gender"`
	Avatar   string      `json:"avatar"`
	Extend   *UserExtend `json:"extend"`
}

type UserExtend struct {
	IsReady  bool `json:"isReady"`
	IsServer bool `json:"isServer"` // 给服务机器人的，看是不是再服务中
}

// game 服务器(特殊用户)
type GameServer struct {
	User
	RoomId string `json:"roomId"`
}

// 客户端
type Client struct {
	User      *User           `json:"user" sm:"用户信息"`
	RoomType  RoomType        `json:"roomType" sm:"房间类型"`
	RoomId    string          `json:"roomId" sm:"房间Id"`
	conn      *websocket.Conn `sm:"websocket.Conn"`
	send      chan []byte     `sm:"异步写队列"`
	done      chan struct{}   `sm:"关闭信号"`
	mu        sync.Mutex      `sm:"保护 conn 置 nil"`
	closeOnce sync.Once
	closed    uint32
}

const (
	maxMessageSize = 512 * 1024
	pingPeriod     = 54 * time.Second
	writeWait      = 10 * time.Second
)

// 安全关闭 client：先标记已关闭，关闭 done、conn，再关闭 send（只做一次）
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		atomic.StoreUint32(&c.closed, 1)
		// 通知 pump 退出
		close(c.done)
		// 先关闭底层连接，触发 readPump/writePump 退出
		_ = c.conn.Close()
		// 关闭 send，writePump 会因为 chan 关闭而退出
		close(c.send)
	})
}

// 判断是否已关闭
func (c *Client) IsClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

// 读泵：只负责读 + 出错时清理
func (c *Client) readPump(ctx context.Context, m *Manager) {
	defer func() {
		m.removeClient(ctx, c.User.Id)
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
		case "createOrJoinMap":
			payload = new(CreateOrJoinMapReq)
		case "JoinRoomHandler":
			payload = new(JoinRoomReq)
		case "action":
			payload = new(ActionReq)
		case "userIsReady":
			payload = nil
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
