// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Users is the golang structure for table users.
type Users struct {
	Id              uint64      `json:"id"              orm:"id"                description:""`                     //
	Account         string      `json:"account"         orm:"account"           description:"账号"`                   // 账号
	Password        string      `json:"password"        orm:"password"          description:""`                     //
	Name            string      `json:"name"            orm:"name"              description:""`                     //
	Phone           string      `json:"phone"           orm:"phone"             description:"手机号码"`                 // 手机号码
	Avatar          string      `json:"avatar"          orm:"avatar"            description:"头像"`                   // 头像
	Gender          uint        `json:"gender"          orm:"gender"            description:"性别 1 男 2 女"`           // 性别 1 男 2 女
	Role            uint        `json:"role"            orm:"role"              description:"角色 1 站长 2 管理员 3 普通人员"` // 角色 1 站长 2 管理员 3 普通人员
	Email           string      `json:"email"           orm:"email"             description:""`                     //
	EmailVerifiedAt *gtime.Time `json:"emailVerifiedAt" orm:"email_verified_at" description:""`                     //
	Signature       string      `json:"signature"       orm:"signature"         description:"签名"`                   // 签名
	RegisterIp      string      `json:"registerIp"      orm:"register_ip"       description:"注册地 ip"`               // 注册地 ip
	RememberToken   string      `json:"rememberToken"   orm:"remember_token"    description:""`                     //
	CreatedAt       *gtime.Time `json:"createdAt"       orm:"created_at"        description:""`                     //
	UpdatedAt       *gtime.Time `json:"updatedAt"       orm:"updated_at"        description:""`                     //
}
