package glasses

import (
	"context"
	"slices"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
)

type sGlasses struct{}

func init() {
	service.RegisterGlasses(New())
}

func New() service.IGlasses {
	return &sGlasses{}
}

func (s *sGlasses) UpdateStatus(ctx context.Context, ids []int64, status uint) (err error) {
	if len(ids) == 0 {
		return gerror.New("ids 不能为空")
	}

	var statusArr = []uint{1, 2}
	if !slices.Contains(statusArr, status) {
		return gerror.New("状态无效")
	}

	_, err = dao.Glasses.Ctx(ctx).Unscoped().WhereIn(dao.Glasses.Columns().Id, ids).Data(do.Glasses{
		Status: status,
	}).Update()
	return nil
}

func (s *sGlasses) UpdateUseStatus(ctx context.Context, ids []int64, useStatus uint) (err error) {
	if len(ids) == 0 {
		return gerror.New("ids 不能为空")
	}

	var statusArr = []uint{1, 2}
	if !slices.Contains(statusArr, useStatus) {
		return gerror.New("状态无效")
	}

	_, err = dao.Glasses.Ctx(ctx).Unscoped().WhereIn(dao.Glasses.Columns().Id, ids).Data(do.Glasses{
		UseStatus: useStatus,
	}).Update()
	return nil
}

func (s *sGlasses) One(ctx context.Context, id int64, status uint) (glasses *model.Glasses, err error) {
	query := dao.Glasses.Ctx(ctx).Where(do.Glasses{
		Id: id,
	})

	if status > 0 {
		query.Where(do.Glasses{
			Status: status,
		})
	}

	err = query.Scan(&glasses)
	return
}
