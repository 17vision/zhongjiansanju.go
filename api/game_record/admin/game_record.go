package admin

import (
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

type InfoReq struct {
	g.Meta `path:"/game_records/info" method:"get" tags:"游戏记录" summary:"获取游戏记录信息" description:"获取游戏记录信息"`
}

type InfoRes struct {
	model.GameRecordInfoRes
}

type TimeperiodCountReq struct {
	g.Meta `path:"/game_records/timeperiod_count" method:"get" tags:"游戏记录" summary:"获取时间段游戏记录数量" description:"获取时间段游戏记录数量"`
	model.GameRecordTimeperiodCountReq
}

type TimeperiodCountRes struct {
	model.GameRecordTimeperiodCountRes
}

type ListReq struct {
	g.Meta `path:"/game_records" method:"get" tags:"游戏记录" summary:"获取游戏记录列表" description:"获取游戏记录列表"`
	model.GameRecordListReq
}

type ListRes struct {
	model.GameRecordListRes
}

// 导出数据
type ExportReq struct {
	g.Meta `path:"/game_records/export" method:"post" tags:"游戏记录" summary:"导出游戏记录" description:"导出游戏记录"`
	model.GameRecordExportReq
}

type ExportRes struct {
	model.GameRecordExportRes
}
