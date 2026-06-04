package admin

import (
	"zjsj/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// 添加
type CreateReq struct {
	g.Meta `path:"/glasses" method:"post" tags:"设备管理" summary:"创建眼镜" description:"创建新眼镜"`

	model.GlassesCreateReq
}

type CreateRes struct {
	model.GlassesCreateRes
}

// 列表
type ListReq struct {
	g.Meta `path:"/glasses" method:"get" tags:"设备管理" summary:"获取眼镜列表" description:"获取眼镜列表"`
	model.GlassesListReq
}

type ListRes struct {
	model.GlassesListRes
}

// 信息
type InfoReq struct {
	g.Meta `path:"/glasses/info" method:"get" tags:"设备管理" summary:"获取眼镜信息" description:"获取眼镜信息"`
}

type InfoRes struct {
	model.GlassesInfoRes
}
