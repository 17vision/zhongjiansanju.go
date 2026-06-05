// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package glasses

import (
	"context"

	"zjsj/api/glasses/admin"
	"zjsj/api/glasses/api"
)

type IGlassesAdmin interface {
	Create(ctx context.Context, req *admin.CreateReq) (res *admin.CreateRes, err error)
	List(ctx context.Context, req *admin.ListReq) (res *admin.ListRes, err error)
	Info(ctx context.Context, req *admin.InfoReq) (res *admin.InfoRes, err error)
	Delete(ctx context.Context, req *admin.DeleteReq) (res *admin.DeleteRes, err error)
}

type IGlassesApi interface {
	Create(ctx context.Context, req *api.CreateReq) (res *api.CreateRes, err error)
	List(ctx context.Context, req *api.ListReq) (res *api.ListRes, err error)
}
