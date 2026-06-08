package game_record

import (
	"context"

	"zjsj/api/game_record/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Info(ctx context.Context, req *admin.InfoReq) (res *admin.InfoRes, err error) {
	data, err := service.GameRecords().Info(ctx)
	if err != nil {
		return
	}
	res = &admin.InfoRes{}
	res.GameRecordInfoRes = *data
	return
}
