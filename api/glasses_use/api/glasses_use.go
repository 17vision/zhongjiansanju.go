package api

import (
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

type CreateReq struct {
	g.Meta `path:"/glasses_use" method:"post" tags:"眼镜使用" summary:"创建眼镜使用记录" description:"创建新眼镜使用记录"`
	model.GlassesUseCreateReq
}

type CreateRes struct {
	model.GlassesUseCreateRes
}

type OneReq struct {
	g.Meta `path:"/glasses_use/one" method:"get" tags:"眼镜使用" summary:"获取眼镜使用记录" description:"获取眼镜使用记录"`
	model.GlassesUseOneReq
}

type OneRes struct {
	model.GlassesUseOneRes
}
