// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package glasses_use

import (
	"context"

	"zjsj/api/glasses_use/api"
)

type IGlassesUseApi interface {
	Create(ctx context.Context, req *api.CreateReq) (res *api.CreateRes, err error)
	One(ctx context.Context, req *api.OneReq) (res *api.OneRes, err error)
}
