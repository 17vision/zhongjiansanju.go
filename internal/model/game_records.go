package model

import "github.com/gogf/gf/v2/os/gtime"

type GameRecordsCreateReq struct {
	GlassesUseId uint64      `json:"glassesUseId" orm:"glasses_use_id" description:"眼镜使用 id"`
	Name         string      `json:"name"         orm:"name"           description:"游戏名称"`
	Ip           string      `json:"ip"           orm:"ip"             description:"客户端 ip"`
	ConnectAt    *gtime.Time `json:"connectAt"    orm:"connect_at"     description:""`
	StartAt      *gtime.Time `json:"startAt"      orm:"start_at"       description:""`
}

type GameRecordsCreateRes struct {
	Id        uint64      `json:"id"`
	CreatedAt *gtime.Time `json:"createdAt" dc:"创建时间"`
}

type GameRecordsUpdateReq struct {
	Id    uint64      `json:"id"           orm:"id"             description:""`
	EndAt *gtime.Time `json:"endAt"        orm:"end_at"         description:""`
}
