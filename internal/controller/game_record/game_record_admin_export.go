package game_record

import (
	"context"

	"zjsj/api/game_record/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Export(ctx context.Context, req *admin.ExportReq) (res *admin.ExportRes, err error) {
	data, err := service.GameRecords().Export(ctx, req.GameRecordExportReq)
	if err != nil {
		return
	}
	res = &admin.ExportRes{}
	res.GameRecordExportRes = data
	return
}
