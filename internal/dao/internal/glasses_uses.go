// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// GlassesUsesDao is the data access object for the table glasses_uses.
type GlassesUsesDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  GlassesUsesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// GlassesUsesColumns defines and stores column names for the table glasses_uses.
type GlassesUsesColumns struct {
	Id          string //
	EquipmentSn string // 设备序列号
	Nickname    string // 昵称
	Model       string // 模型
	Status      string // 状态 1 待使用 2 使用中 3 已使用 4 已作废
	CreatedAt   string //
	UpdatedAt   string //
}

// glassesUsesColumns holds the columns for the table glasses_uses.
var glassesUsesColumns = GlassesUsesColumns{
	Id:          "id",
	EquipmentSn: "equipment_sn",
	Nickname:    "nickname",
	Model:       "model",
	Status:      "status",
	CreatedAt:   "created_at",
	UpdatedAt:   "updated_at",
}

// NewGlassesUsesDao creates and returns a new DAO object for table data access.
func NewGlassesUsesDao(handlers ...gdb.ModelHandler) *GlassesUsesDao {
	return &GlassesUsesDao{
		group:    "default",
		table:    "glasses_uses",
		columns:  glassesUsesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *GlassesUsesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *GlassesUsesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *GlassesUsesDao) Columns() GlassesUsesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *GlassesUsesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *GlassesUsesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *GlassesUsesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
