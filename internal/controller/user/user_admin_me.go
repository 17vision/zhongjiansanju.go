package user

import (
	"context"

	"zjsj/api/user/admin"
	"zjsj/internal/pkg/jwt"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/util/gconv"
)

func (c *ControllerAdmin) Me(ctx context.Context, req *admin.MeReq) (res *admin.MeRes, err error) {
	id := jwt.NewJwt(ctx).GetIdentity(ctx)

	data, err := service.User().Me(ctx, gconv.Int64(id))
	if err != nil {
		return nil, err
	}

	res = &admin.MeRes{
		User: *data,
	}
	return res, nil
}
