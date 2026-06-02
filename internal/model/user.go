package model

import (
	"github.com/gogf/gf/v2/os/gtime"
)

type User struct {
	Id        uint64      `json:"id"`
	Account   string      `json:"account"`
	Password  string      `json:"password,omitempty"`
	Name      string      `json:"name"`
	Phone     string      `json:"phone"`
	Avatar    string      `json:"avatar"`
	Gender    uint        `json:"gender"`
	Role      uint        `json:"role"`
	Email     string      `json:"email"`
	Signature string      `json:"signature"`
	RoleStr   string      `json:"roleStr"`
	GenderStr string      `json:"genderStr"`
	CreatedAt *gtime.Time `json:"createdAt"`
}

type UserLoginRes struct {
	Token     string      `json:"token"`
	ExpiredAt *gtime.Time `json:"expiredAt"`
}

type UserListReq struct {
	PaginateReq
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type UserListRes struct {
	PaginateRes
	Data []*User `json:"data"`
}

type UserCreateReq struct {
	Account  string `json:"account" v:"required#账号不能为空|max-length:20#账号长度不能超过20个字符"`
	Password string `json:"password" v:"required#密码不能为空|max-length:20#密码长度不能超过20个字符|min-length:6#密码长度不能小于6个字符"`
	Name     string `json:"name" v:"required#姓名不能为空"`
	Gender   int    `json:"gender" v:"required#性别不能为空|in:1,2#性别只能是1、2"`
	Role     int    `json:"role" v:"required#角色不能为空|in:2,3#角色只能是2、3"`
}

type UserCreateRes struct {
	User
}

type UserUpdateReq struct {
	Password string `json:"password" v:"max-length:20#密码长度不能超过20个字符|min-length:6#密码长度不能小于6个字符|required-without-all:name,gender,role"`
	Name     string `json:"name" v:"required-without-all:password,gender,role"`
	Gender   int    `json:"gender" v:"in:1,2#性别只能是1、2|required-without-all:password,name,role"`
	Role     int    `json:"role" v:"in:1,2,3#角色只能是1、2、3|required-without-all:password,name,gender"`
}

type UserUpdateRes struct {
	User
}
