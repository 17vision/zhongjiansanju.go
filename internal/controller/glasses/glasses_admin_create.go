package glasses

import (
	"context"

	"zjsj/api/glasses/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Create(ctx context.Context, req *admin.CreateReq) (res *admin.CreateRes, err error) {
	data, err := service.Glasses().Create(ctx, &req.GlassesCreateReq)
	if err != nil {
		return nil, err
	}

	res = &admin.CreateRes{
		GlassesCreateRes: *data,
	}
	return
}
