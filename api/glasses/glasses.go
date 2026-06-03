// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package glasses

import (
	"context"

	"zjsj/api/glasses/api"
)

type IGlassesApi interface {
	Create(ctx context.Context, req *api.CreateReq) (res *api.CreateRes, err error)
}
