// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// GlassesUses is the golang structure for table glasses_uses.
type GlassesUses struct {
	Id        uint64      `json:"id"        orm:"id"         description:""`                           //
	GlassesId uint64      `json:"glassesId" orm:"glasses_id" description:"眼镜 id"`                      // 眼镜 id
	Nickname  string      `json:"nickname"  orm:"nickname"   description:"昵称"`                         // 昵称
	Model     string      `json:"model"     orm:"model"      description:"模型"`                         // 模型
	Status    uint        `json:"status"    orm:"status"     description:"状态 1 待使用 2 使用中 3 已使用 4 已作废"` // 状态 1 待使用 2 使用中 3 已使用 4 已作废
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""`                           //
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:""`                           //
}
