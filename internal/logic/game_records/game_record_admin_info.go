package gamerecords

import (
	"context"
	"math"
	"time"
	"zjsj/internal/dao"
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/os/gtime"
)

func (s *sGameRecords) Info(ctx context.Context) (res *model.GameRecordInfoRes, err error) {
	now := gtime.Now()
	todayStart := now.StartOfDay()
	weekStart := now.StartOfWeek()

	yesterday := now.Add(-24 * time.Hour)
	yesterdayStart := yesterday.StartOfDay()
	yesterdayEnd := yesterday.EndOfDay()

	columns := dao.GameRecords.Columns()

	res = &model.GameRecordInfoRes{}

	res.TodayCount, err = dao.GameRecords.Ctx(ctx).WhereGT(columns.CreatedAt, todayStart).Count()
	if err != nil {
		return res, err
	}

	res.YesterdayCount, err = dao.GameRecords.Ctx(ctx).WhereBetween(columns.CreatedAt, yesterdayStart, yesterdayEnd).Count()
	if err != nil {
		return res, err
	}

	res.WeekTotal, err = dao.GameRecords.Ctx(ctx).WhereGT(columns.CreatedAt, weekStart).Count()
	if err != nil {
		return res, err
	}

	res.GrowthFromYesterday = res.TodayCount - res.YesterdayCount

	if res.YesterdayCount != 0 {
		res.GrowthRate = math.Round(float64(res.GrowthFromYesterday)/float64(res.YesterdayCount)*1000) / 10
	} else {
		res.GrowthRate = 0
	}
	return
}
