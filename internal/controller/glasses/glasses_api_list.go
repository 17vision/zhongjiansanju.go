package glasses

import (
	"context"

	"zjsj/api/glasses/api"
	"zjsj/internal/service"
)

func (c *ControllerApi) List(ctx context.Context, req *api.ListReq) (res *api.ListRes, err error) {
	data, err := service.Glasses().List(ctx, req.GlassesListReq)
	if err != nil {
		return
	}

	res = &api.ListRes{}
	res.GlassesListRes = *data
	return
}
