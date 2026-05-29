package admin

import (
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

type LoginReq struct {
	g.Meta   `path:"/login" method:"post" summary:"登录"`
	Account  string `json:"account" v:"required#账号不能为空|max-length:20#账号长度不能超过20个字符"`
	Password string `json:"password" v:"required#密码不能为空|max-length:20#密码长度不能超过20个字符|min-length:6#密码长度不能小于6个字符"`
}

type LoginRes struct {
	model.UserLoginRes
}
