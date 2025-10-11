package websocket

import "github.com/gorilla/websocket"

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

type Client struct {
	User           *User           `json:"user" sm:"用户信息"`
	Conn           *websocket.Conn `json:"conn" sm:"websocket.Conn"`
	RoomId         string          `json:"roomId" sm:"房间Id"`
	LastHeartbeat  int64           `json:"lastHeartbeat" sm:"最后心跳时间戳"`
	ReconnectCount int             `json:"reconnectCount" sm:"断线重连次数"`
}

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
