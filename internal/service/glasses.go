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
		UpdateStatus(ctx context.Context, ids []int64, status uint) (err error)
		UpdateUseStatus(ctx context.Context, ids []int64, useStatus uint) (err error)
		One(ctx context.Context, id int64, status uint) (glasses *model.Glasses, err error)
		Delete(ctx context.Context, req model.GlassesDeleteReq) (res *model.Result, err error)
		List(ctx context.Context, req model.GlassesListReq) (res *model.GlassesListRes, err error)
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
