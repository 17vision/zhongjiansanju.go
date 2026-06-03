package glasses

import (
	"context"

	"zjsj/api/glasses/api"
	"zjsj/internal/service"
)

func (c *ControllerApi) Create(ctx context.Context, req *api.CreateReq) (res *api.CreateRes, err error) {
	data, err := service.Glasses().Create(ctx, &req.GlassesCreateReq)
	if err != nil {
		return nil, err
	}

	res = &api.CreateRes{
		GlassesCreateRes: *data,
	}
	return
}
