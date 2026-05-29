package model

import (
	"github.com/gogf/gf/v2/os/gtime"
)

type UserLoginRes struct {
	Token     string      `json:"token"`
	ExpireIn  *gtime.Time `json:"expireIn"`
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
}
