// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// GlassesUses is the golang structure of table glasses_uses for DAO operations like Where/Data.
type GlassesUses struct {
	g.Meta    `orm:"table:glasses_uses, do:true"`
	Id        any         //
	GlassesId any         // 眼镜 id
	Nickname  any         // 昵称
	Model     any         // 模型
	Status    any         // 状态 1 待使用 2 使用中 3 已使用 4 已作废
	CreatedAt *gtime.Time //
	UpdatedAt *gtime.Time //
}
