// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// UsersDao is the data access object for the table users.
type UsersDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  UsersColumns       // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// UsersColumns defines and stores column names for the table users.
type UsersColumns struct {
	Id              string //
	Account         string // 账号
	Password        string //
	Name            string //
	Phone           string // 手机号码
	Avatar          string // 头像
	Gender          string // 性别 1 男 2 女
	Role            string // 角色 1 站长 2 管理员 3 普通人员
	Email           string //
	EmailVerifiedAt string //
	Signature       string // 签名
	RegisterIp      string // 注册地 ip
	RememberToken   string //
	CreatedAt       string //
	UpdatedAt       string //
}

// usersColumns holds the columns for the table users.
var usersColumns = UsersColumns{
	Id:              "id",
	Account:         "account",
	Password:        "password",
	Name:            "name",
	Phone:           "phone",
	Avatar:          "avatar",
	Gender:          "gender",
	Role:            "role",
	Email:           "email",
	EmailVerifiedAt: "email_verified_at",
	Signature:       "signature",
	RegisterIp:      "register_ip",
	RememberToken:   "remember_token",
	CreatedAt:       "created_at",
	UpdatedAt:       "updated_at",
}

// NewUsersDao creates and returns a new DAO object for table data access.
func NewUsersDao(handlers ...gdb.ModelHandler) *UsersDao {
	return &UsersDao{
		group:    "default",
		table:    "users",
		columns:  usersColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *UsersDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *UsersDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *UsersDao) Columns() UsersColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *UsersDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *UsersDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *UsersDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
