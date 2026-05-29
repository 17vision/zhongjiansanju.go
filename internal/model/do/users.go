// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Users is the golang structure of table users for DAO operations like Where/Data.
type Users struct {
	g.Meta          `orm:"table:users, do:true"`
	Id              any         //
	Account         any         // 账号
	Password        any         //
	Name            any         //
	Phone           any         // 手机号码
	Avatar          any         // 头像
	Gender          any         // 性别 1 男 2 女
	Role            any         // 角色 1 站长 2 管理员 3 普通人员
	Email           any         //
	EmailVerifiedAt *gtime.Time //
	Signature       any         // 签名
	RegisterIp      any         // 注册地 ip
	RememberToken   any         //
	CreatedAt       *gtime.Time //
	UpdatedAt       *gtime.Time //
}
