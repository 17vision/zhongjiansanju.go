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
	IGameRecords interface {
		Info(ctx context.Context) (res *model.GameRecordInfoRes, err error)
		List(ctx context.Context, req model.GameRecordListReq) (res *model.GameRecordListRes, err error)
		TimeperiodCount(ctx context.Context, req model.GameRecordTimeperiodCountReq) (res *model.GameRecordTimeperiodCountRes, err error)
		Create(ctx context.Context, req model.GameRecordsCreateReq) (res *model.GameRecordsCreateRes, err error)
		Update(ctx context.Context, req model.GameRecordsUpdateReq) (err error)
	}
)

var (
	localGameRecords IGameRecords
)

func GameRecords() IGameRecords {
	if localGameRecords == nil {
		panic("implement not found for interface IGameRecords, forgot register?")
	}
	return localGameRecords
}

func RegisterGameRecords(i IGameRecords) {
	localGameRecords = i
}
