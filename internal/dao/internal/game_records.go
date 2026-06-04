// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// GameRecordsDao is the data access object for the table game_records.
type GameRecordsDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  GameRecordsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// GameRecordsColumns defines and stores column names for the table game_records.
type GameRecordsColumns struct {
	Id           string //
	GlassesUseId string // 眼镜使用 id
	Name         string // 游戏名称
	Ip           string // 客户端 ip
	ConnectAt    string //
	StartAt      string //
	EndAt        string //
	CreatedAt    string //
	UpdatedAt    string //
}

// gameRecordsColumns holds the columns for the table game_records.
var gameRecordsColumns = GameRecordsColumns{
	Id:           "id",
	GlassesUseId: "glasses_use_id",
	Name:         "name",
	Ip:           "ip",
	ConnectAt:    "connect_at",
	StartAt:      "start_at",
	EndAt:        "end_at",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
}

// NewGameRecordsDao creates and returns a new DAO object for table data access.
func NewGameRecordsDao(handlers ...gdb.ModelHandler) *GameRecordsDao {
	return &GameRecordsDao{
		group:    "default",
		table:    "game_records",
		columns:  gameRecordsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *GameRecordsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *GameRecordsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *GameRecordsDao) Columns() GameRecordsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *GameRecordsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *GameRecordsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *GameRecordsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
