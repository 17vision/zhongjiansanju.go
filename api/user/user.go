// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package user

import (
	"context"

	"zjsj/api/user/admin"
)

type IUserAdmin interface {
	Login(ctx context.Context, req *admin.LoginReq) (res *admin.LoginRes, err error)
	Me(ctx context.Context, req *admin.MeReq) (res *admin.MeRes, err error)
	UserList(ctx context.Context, req *admin.UserListReq) (res *admin.UserListRes, err error)
	Create(ctx context.Context, req *admin.CreateReq) (res *admin.CreateRes, err error)
	Update(ctx context.Context, req *admin.UpdateReq) (res *admin.UpdateRes, err error)
}
