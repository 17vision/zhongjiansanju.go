// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// GameRecords is the golang structure of table game_records for DAO operations like Where/Data.
type GameRecords struct {
	g.Meta       `orm:"table:game_records, do:true"`
	Id           any         //
	GlassesUseId any         // 眼镜使用 id
	Name         any         // 游戏名称
	Ip           any         // 客户端 ip
	ConnectAt    *gtime.Time //
	StartAt      *gtime.Time //
	EndAt        *gtime.Time //
	CreatedAt    *gtime.Time //
	UpdatedAt    *gtime.Time //
}
