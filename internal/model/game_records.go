package model

import (
	"zjsj/internal/model/entity"

	"github.com/gogf/gf/v2/os/gtime"
)

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

type GameRecordInfoRes struct {
	TodayCount          int     `json:"todayCount" dc:"今日体验次数"`
	YesterdayCount      int     `json:"yesterdayCount" dc:"昨日体验次数"`
	WeekTotal           int     `json:"weekTotal" dc:"本周体验总次数"`
	GrowthFromYesterday int     `json:"growthFromYesterday" dc:"较昨日增长次数"`
	GrowthRate          float64 `json:"growthRate" dc:"较昨日增长率"`
}

type GameRecordTimeperiodCount struct {
	Hour  string `json:"hour"`
	Count int    `json:"count"`
}

type GameRecordTimeperiodCountReq struct {
	StartAt *gtime.Time `json:"startAt" v:"required#开始时间不能为空" dc:"开始时间" `
	EndAt   *gtime.Time `json:"endAt" v:"required#结束时间不能为空|gt:StartAt#结束时间必须大于开始时间"  dc:"结束时间"`
}

type GameRecordTimeperiodCountRes struct {
	List    []GameRecordTimeperiodCount `json:"list"`
	Total   int                         `json:"total"`
	StartAt *gtime.Time                 `json:"startAt" dc:"开始时间"`
	EndAt   *gtime.Time                 `json:"endAt" dc:"结束时间"`
}

type GlassesUseWithGlasses struct {
	*entity.GlassesUses
	Glasses entity.Glasses `json:"glasses" orm:"with:id=glasses_id"`
}

func (GlassesUseWithGlasses) TableName() string {
	return "glasses_uses"
}

type GameRecord struct {
	*entity.GameRecords
	GlassesUse GlassesUseWithGlasses `json:"glassesUse" orm:"with:id=glasses_use_id"`
}

type GameRecordListReq struct {
	PaginateReq
	StartAt *gtime.Time `json:"startAt" v:"required_with:EndAt" dc:"开始时间" `
	EndAt   *gtime.Time `json:"endAt" v:"required_with:StartAt"  dc:"结束时间"`
}

type GameRecordListRes struct {
	PaginateRes
	Data []*GameRecord `json:"data"`
}

type GameRecordExportReq struct {
	StartAt *gtime.Time `json:"startAt" dc:"开始时间"`
	EndAt   *gtime.Time `json:"endAt" dc:"结束时间"`
}

type GameRecordExportRes struct {
	Total int `json:"total"`
}
