package glasses_use

import (
	"context"

	"zjsj/api/glasses_use/api"
	"zjsj/internal/service"
)

func (c *ControllerApi) One(ctx context.Context, req *api.OneReq) (res *api.OneRes, err error) {
	data, err := service.GlassesUse().One(ctx, req.EquipmentSn)
	if err != nil {
		return
	}

	res = &api.OneRes{
		GlassesUseOneRes: *data,
	}
	return
}
