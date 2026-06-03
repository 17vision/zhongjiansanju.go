package model

import "github.com/gogf/gf/v2/os/gtime"

type GlassesUseCreateReq struct {
	EquipmentSn string `json:"equipmentSn" v:"required|length:1,32#设备序列号不能为空|序列号长度不能超过32字符" dc:"设备唯一序列号"`
	Nickname    string `json:"nickname" v:"required|length:1,16#昵称不能为空|昵称长度不能超过16字符" dc:"昵称"`
	Model       string `json:"model" v:"required|length:1,32#模型名称不能为空|模型名称长度不能超过32字符" dc:"模型名称"`
}

type GlassesUseCreateRes struct {
	Id        uint64      `json:"id" dc:"使用记录ID"`
	CreatedAt *gtime.Time `json:"createdAt" dc:"创建时间"`
}
