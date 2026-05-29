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
}
