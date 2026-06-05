package glasses

import (
	"context"

	"zjsj/api/glasses/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Delete(ctx context.Context, req *admin.DeleteReq) (res *admin.DeleteRes, err error) {
	data, err := service.Glasses().Delete(ctx, req.GlassesDeleteReq)
	if err != nil {
		return nil, err
	}

	res = &admin.DeleteRes{
		Result: *data,
	}
	return
}
