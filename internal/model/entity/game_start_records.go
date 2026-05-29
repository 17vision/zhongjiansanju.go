// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// GameStartRecords is the golang structure for table game_start_records.
type GameStartRecords struct {
	Id        uint64      `json:"id"        orm:"id"         description:""`       //
	Name      string      `json:"name"      orm:"name"       description:"游戏名称"`   // 游戏名称
	Sn        string      `json:"sn"        orm:"sn"         description:"设备 sn"`  // 设备 sn
	Ip        string      `json:"ip"        orm:"ip"         description:"客户端 ip"` // 客户端 ip
	ConnectAt *gtime.Time `json:"connectAt" orm:"connect_at" description:""`       //
	StartAt   *gtime.Time `json:"startAt"   orm:"start_at"   description:""`       //
	EndAt     *gtime.Time `json:"endAt"     orm:"end_at"     description:""`       //
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""`       //
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:""`       //
}
