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
	IGlasses interface {
		Create(ctx context.Context, req *model.GlassesCreateReq) (res *model.GlassesCreateRes, err error)
	}
)

var (
	localGlasses IGlasses
)

func Glasses() IGlasses {
	if localGlasses == nil {
		panic("implement not found for interface IGlasses, forgot register?")
	}
	return localGlasses
}

func RegisterGlasses(i IGlasses) {
	localGlasses = i
}
