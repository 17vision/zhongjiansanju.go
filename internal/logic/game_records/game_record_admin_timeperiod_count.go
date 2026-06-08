package gamerecords

import (
	"context"
	"fmt"
	"zjsj/internal/dao"
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *sGameRecords) TimeperiodCount(ctx context.Context, req model.GameRecordTimeperiodCountReq) (res *model.GameRecordTimeperiodCountRes, err error) {
	var dbResult []model.GameRecordTimeperiodCount
	err = dao.GameRecords.Ctx(ctx).
		WhereBetween(dao.GameRecords.Columns().StartAt, req.StartAt, req.EndAt).
		Fields("DATE_FORMAT(start_at, '%H:00') as hour, COUNT(*) as count").
		Group("hour").
		Order("hour").
		Scan(&dbResult)

	if err != nil {
		return nil, gerror.Wrap(err, "统计游戏时段数据失败")
	}

	countMap := make(map[string]int)
	for _, item := range dbResult {
		countMap[item.Hour] = item.Count
	}

	fullList := generate24HourList()

	var total int
	for i := range fullList {
		if count, exists := countMap[fullList[i].Hour]; exists {
			fullList[i].Count = count
			total += count
		}
	}

	return &model.GameRecordTimeperiodCountRes{
		List:    fullList,
		Total:   total,
		StartAt: req.StartAt,
		EndAt:   req.EndAt,
	}, nil
}

func generate24HourList() []model.GameRecordTimeperiodCount {
	var list []model.GameRecordTimeperiodCount
	for hour := 9; hour < 24; hour++ {
		list = append(list, model.GameRecordTimeperiodCount{
			Hour:  fmt.Sprintf("%02d:00", hour), // 格式化为两位数字：00:00, 01:00 ... 23:00
			Count: 0,
		})
	}
	return list
}
