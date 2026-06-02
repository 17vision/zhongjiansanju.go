// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"zjsj/internal/model"
)

type (
	IUser interface {
		AccountExist(ctx context.Context, account string) (res bool, err error)
		Create(ctx context.Context, req *model.UserCreateReq) (res *model.UserCreateRes, err error)
		Login(ctx context.Context, account string, password string) (res *model.UserLoginRes, err error)
		Me(ctx context.Context, id int64) (res *model.User, err error)
		Update(ctx context.Context, id int64, req *model.UserUpdateReq) (res *model.UserUpdateRes, err error)
		List(ctx context.Context, req *model.UserListReq) (res *model.UserListRes, err error)
	}
)

var (
	localUser IUser
)

func User() IUser {
	if localUser == nil {
		panic("implement not found for interface IUser, forgot register?")
	}
	return localUser
}

func RegisterUser(i IUser) {
	localUser = i
}
