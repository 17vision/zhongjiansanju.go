package user

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"

	"zjsj/api/user/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Create(ctx context.Context, req *admin.CreateReq) (res *admin.CreateRes, err error) {
	exists, err := service.User().AccountExist(ctx, req.Account)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, gerror.New("该账号已存在")
	}

	data, err := service.User().Create(ctx, &req.UserCreateReq)
	if err != nil {
		return nil, err
	}
	return &admin.CreateRes{UserCreateRes: *data}, nil
}
