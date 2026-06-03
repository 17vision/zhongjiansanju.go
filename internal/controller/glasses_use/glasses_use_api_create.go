package glasses_use

import (
	"context"

	"zjsj/api/glasses_use/api"
	"zjsj/internal/service"
)

func (c *ControllerApi) Create(ctx context.Context, req *api.CreateReq) (res *api.CreateRes, err error) {
	data, err := service.GlassesUse().Create(ctx, &req.GlassesUseCreateReq)
	if err != nil {
		return
	}

	res = &api.CreateRes{
		GlassesUseCreateRes: *data,
	}
	return
}
