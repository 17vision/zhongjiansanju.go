package game_record

import (
	"context"

	"zjsj/api/game_record/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) TimeperiodCount(ctx context.Context, req *admin.TimeperiodCountReq) (res *admin.TimeperiodCountRes, err error) {
	data, err := service.GameRecords().TimeperiodCount(ctx, req.GameRecordTimeperiodCountReq)
	if err != nil {
		return
	}

	res = &admin.TimeperiodCountRes{}
	res.GameRecordTimeperiodCountRes = *data
	return
}
