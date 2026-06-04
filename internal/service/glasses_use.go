// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"zjsj/internal/model"
)

type (
	IGlassesUse interface {
		UpdateStatus(ctx context.Context, id int64, status uint) (err error)
		Create(ctx context.Context, req *model.GlassesUseCreateReq) (res *model.GlassesUseCreateRes, err error)
		One(ctx context.Context, equipmentSn string) (res *model.GlassesUseOneRes, err error)
	}
)

var (
	localGlassesUse IGlassesUse
)

func GlassesUse() IGlassesUse {
	if localGlassesUse == nil {
		panic("implement not found for interface IGlassesUse, forgot register?")
	}
	return localGlassesUse
}

func RegisterGlassesUse(i IGlassesUse) {
	localGlassesUse = i
}
