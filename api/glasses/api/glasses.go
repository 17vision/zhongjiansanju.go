package api

import (
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

type CreateReq struct {
	g.Meta `path:"/glasses" method:"post" tags:"设备管理" summary:"创建眼镜" description:"创建新眼镜"`

	model.GlassesCreateReq
}

type CreateRes struct {
	model.GlassesCreateRes
}
