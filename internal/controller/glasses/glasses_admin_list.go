package glasses

import (
	"context"

	"zjsj/api/glasses/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) List(ctx context.Context, req *admin.ListReq) (res *admin.ListRes, err error) {
	data, err := service.Glasses().List(ctx, req.GlassesListReq)
	if err != nil {
		return
	}

	res = &admin.ListRes{}
	res.GlassesListRes = *data
	return
}
