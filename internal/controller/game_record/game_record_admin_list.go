package game_record

import (
	"context"

	"zjsj/api/game_record/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) List(ctx context.Context, req *admin.ListReq) (res *admin.ListRes, err error) {
	data, err := service.GameRecords().List(ctx, req.GameRecordListReq)
	if err != nil {
		return nil, err
	}

	res = &admin.ListRes{
		GameRecordListRes: *data,
	}
	return
}
