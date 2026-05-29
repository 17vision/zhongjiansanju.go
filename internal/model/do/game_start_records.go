// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// GameStartRecords is the golang structure of table game_start_records for DAO operations like Where/Data.
type GameStartRecords struct {
	g.Meta    `orm:"table:game_start_records, do:true"`
	Id        any         //
	Name      any         // 游戏名称
	Sn        any         // 设备 sn
	Ip        any         // 客户端 ip
	ConnectAt *gtime.Time //
	StartAt   *gtime.Time //
	EndAt     *gtime.Time //
	CreatedAt *gtime.Time //
	UpdatedAt *gtime.Time //
}
