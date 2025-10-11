package websocket

import (
	"time"
)

// 心跳检测相关配置
const (
	HeartbeatInterval = 10 * time.Second // 心跳检测间隔
	HeartbeatTimeout  = 30 * time.Second // 心跳超时时间
	MaxReconnect      = 3                // 最大断线重连次数
)

// HeartbeatChecker 用于心跳检测和断线重连控制
// client: 需要检测的客户端，closeFunc: 超时后关闭连接的回调
func HeartbeatChecker(client *Client, closeFunc func()) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		if client == nil {
			return
		}
		if time.Now().UnixMilli()-client.LastHeartbeat > HeartbeatTimeout.Milliseconds() {
			closeFunc()
			return
		}
	}
}
