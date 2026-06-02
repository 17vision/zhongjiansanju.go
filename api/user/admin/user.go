package admin

import (
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// 登录
type LoginReq struct {
	g.Meta   `path:"/login" method:"post" tags:"用户" summary:"登录" description:"后台登录"`
	Account  string `json:"account" v:"required#账号不能为空|max-length:20#账号长度不能超过20个字符"`
	Password string `json:"password" v:"required#密码不能为空|max-length:20#密码长度不能超过20个字符|min-length:6#密码长度不能小于6个字符"`
}

type LoginRes struct {
	model.UserLoginRes
	model.PaginateReq
}

// 获取当前用户数据
type MeReq struct {
	g.Meta `path:"/me" method:"get" tags:"用户" summary:"获取当前用户数据" description:"获取当前登录用户的数据"`
}

type MeRes struct {
	model.User
}

// 用户列表
type UserListReq struct {
	g.Meta `path:"/users" method:"get" tags:"用户" summary:"用户列表" description:"获取用户列表"`
	model.UserListReq
}

type UserListRes struct {
	model.UserListRes
}

// 添加用户
type CreateReq struct {
	g.Meta `path:"/users" method:"post" tags:"用户" summary:"创建用户" description:"创建新用户"`
	model.UserCreateReq
}

type CreateRes struct {
	model.UserCreateRes
}

// 修改用户
type UpdateReq struct {
	g.Meta `path:"/users" method:"put" tags:"用户" summary:"修改用户" description:"修改用户信息"`
	Id     int64 `json:"id" v:"required#用户ID不能为空"`
	model.UserUpdateReq
}

type UpdateRes struct {
	model.UserUpdateRes
}
