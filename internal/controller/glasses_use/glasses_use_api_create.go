package glasses_use

import (
	"context"

	"zjsj/api/glasses_use/api"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (c *ControllerApi) Create(ctx context.Context, req *api.CreateReq) (res *api.CreateRes, err error) {
	glasses, err := service.Glasses().One(ctx, req.GlassesId, 0)
	if err != nil {
		return
	}

	if glasses == nil {
		return nil, gerror.New("设备 id 错误")
	}

	data, err := service.GlassesUse().Create(ctx, &req.GlassesUseCreateReq)
	if err != nil {
		return
	}

	res = &api.CreateRes{
		GlassesUseCreateRes: *data,
	}
	return
}
