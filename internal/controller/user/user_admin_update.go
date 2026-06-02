package user

import (
	"context"

	"zjsj/api/user/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) Update(ctx context.Context, req *admin.UpdateReq) (res *admin.UpdateRes, err error) {
	data, err := service.User().Update(ctx, req.Id, &req.UserUpdateReq)
	if err != nil {
		return nil, err
	}

	return &admin.UpdateRes{UserUpdateRes: *data}, nil
}
