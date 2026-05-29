package user

import (
	"context"

	"zjsj/api/user/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Login(ctx context.Context, req *admin.LoginReq) (res *admin.LoginRes, err error) {
	data, err := service.User().Login(ctx, req.Account, req.Password)
	if err != nil {
		return nil, err
	}

	res = &admin.LoginRes{
		UserLoginRes: *data,
	}
	return
}
