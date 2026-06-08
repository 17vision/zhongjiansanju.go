// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package game_record

import (
	"context"

	"zjsj/api/game_record/admin"
)

type IGameRecordAdmin interface {
	Info(ctx context.Context, req *admin.InfoReq) (res *admin.InfoRes, err error)
	TimeperiodCount(ctx context.Context, req *admin.TimeperiodCountReq) (res *admin.TimeperiodCountRes, err error)
	List(ctx context.Context, req *admin.ListReq) (res *admin.ListRes, err error)
}
