// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// GameStartRecordsDao is the data access object for the table game_start_records.
type GameStartRecordsDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  GameStartRecordsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// GameStartRecordsColumns defines and stores column names for the table game_start_records.
type GameStartRecordsColumns struct {
	Id        string //
	Name      string // 游戏名称
	Sn        string // 设备 sn
	Ip        string // 客户端 ip
	ConnectAt string //
	StartAt   string //
	EndAt     string //
	CreatedAt string //
	UpdatedAt string //
}

// gameStartRecordsColumns holds the columns for the table game_start_records.
var gameStartRecordsColumns = GameStartRecordsColumns{
	Id:        "id",
	Name:      "name",
	Sn:        "sn",
	Ip:        "ip",
	ConnectAt: "connect_at",
	StartAt:   "start_at",
	EndAt:     "end_at",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
}

// NewGameStartRecordsDao creates and returns a new DAO object for table data access.
func NewGameStartRecordsDao(handlers ...gdb.ModelHandler) *GameStartRecordsDao {
	return &GameStartRecordsDao{
		group:    "default",
		table:    "game_start_records",
		columns:  gameStartRecordsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *GameStartRecordsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *GameStartRecordsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *GameStartRecordsDao) Columns() GameStartRecordsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *GameStartRecordsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *GameStartRecordsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *GameStartRecordsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
