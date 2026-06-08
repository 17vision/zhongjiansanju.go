package gamerecords

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
)

func (s *sGameRecords) List(ctx context.Context, req model.GameRecordListReq) (res *model.GameRecordListRes, err error) {
	columns := dao.GameRecords.Columns()

	query := dao.GameRecords.Ctx(ctx).OrderDesc(columns.Id)

	if req.StartAt != nil && req.EndAt != nil {
		query = query.WhereBetween(columns.StartAt, req.StartAt, req.EndAt)
	}

	var data []*model.GameRecord
	var total int
	if err = query.WithAll().Page(req.Page, req.PageSize).ScanAndCount(&data, &total, true); err != nil {
		return nil, err
	}

	res = &model.GameRecordListRes{}
	res.Total = total
	res.Page = req.Page
	res.PageSize = req.PageSize
	res.Data = data
	return
}
