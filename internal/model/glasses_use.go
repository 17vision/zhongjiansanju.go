package model

import (
	"github.com/gogf/gf/v2/os/gtime"
)

type GlassesUse struct {
	Id        uint64      `json:"id"        orm:"id"         description:""`
	GlassesId uint64      `json:"glassesId" orm:"glasses_id" description:"眼镜 id"`
	Nickname  string      `json:"nickname"  orm:"nickname"   description:"昵称"`
	Model     string      `json:"model"     orm:"model"      description:"模型"`
	Status    uint        `json:"status"    orm:"status"     description:"状态 1 待使用 2 使用中 3 已使用 4 已作废"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:""`
}

type GlassesUseCreateReq struct {
	GlassesId int64  `json:"glassesId" v:"required#设备id不能为空" dc:"设备 id"`
	Nickname  string `json:"nickname" v:"required|length:1,16#昵称不能为空|昵称长度不能超过16字符" dc:"昵称"`
	Model     string `json:"model" v:"required|length:1,32#模型名称不能为空|模型名称长度不能超过32字符" dc:"模型名称"`
}

type GlassesUseCreateRes struct {
	Id        uint64      `json:"id" dc:"使用记录ID"`
	CreatedAt *gtime.Time `json:"createdAt" dc:"创建时间"`
}

type GlassesUseOneReq struct {
	EquipmentSn string `json:"equipmentSn" v:"required|length:1,50#设备序列号不能为空|序列号长度不能超过50字符" dc:"设备唯一序列号"`
}

type GlassesUseOneRes struct {
	GlassesUse *GlassesUse `json:"glassesUse"`
	Glasses    *Glasses    `json:"glasses"`
}
