package model

import "github.com/gogf/gf/v2/os/gtime"

type GlassesUseCreateReq struct {
	GlassesId int64  `json:"glassesId" v:"required#设备id不能为空" dc:"设备 id"`
	Nickname  string `json:"nickname" v:"required|length:1,16#昵称不能为空|昵称长度不能超过16字符" dc:"昵称"`
	Model     string `json:"model" v:"required|length:1,32#模型名称不能为空|模型名称长度不能超过32字符" dc:"模型名称"`
}

type GlassesUseCreateRes struct {
	Id        uint64      `json:"id" dc:"使用记录ID"`
	CreatedAt *gtime.Time `json:"createdAt" dc:"创建时间"`
}
