package user

import (
	"context"

	"zjsj/api/user/admin"
	"zjsj/internal/service"
)

func (c *ControllerAdmin) UserList(ctx context.Context, req *admin.UserListReq) (res *admin.UserListRes, err error) {
	data, err := service.User().List(ctx, &req.UserListReq)
	if err != nil {
		return
	}

	res = &admin.UserListRes{}
	res.UserListRes = *data
	return
}
