package glasses_use

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model/do"
	"zjsj/internal/service"
)

type sGlassesUse struct{}

func init() {
	service.RegisterGlassesUse(New())
}

func New() service.IGlassesUse {
	return &sGlassesUse{}
}

func (s *sGlassesUse) UpdateStatus(ctx context.Context, id int64, status uint) (err error) {
	_, err = dao.GlassesUses.Ctx(ctx).WherePri(id).Data(do.GlassesUses{
		Status: status,
	}).Update()

	return
}
