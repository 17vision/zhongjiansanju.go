package glasses_use

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model/do"
	"zjsj/internal/model/entity"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

type sGlassesUse struct{}

func init() {
	service.RegisterGlassesUse(New())
}

func New() service.IGlassesUse {
	return &sGlassesUse{}
}

func (s *sGlassesUse) UpdateStatus(ctx context.Context, id int64, status uint) (err error) {
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		var glassesUses *entity.GlassesUses
		err := dao.GlassesUses.Ctx(ctx).WherePri(id).Scan(&glassesUses)
		if err != nil {
			return err
		}

		if glassesUses == nil {
			return gerror.New("使用设备不存在，请联系管理员")
		}

		res, err := dao.GlassesUses.Ctx(ctx).WherePri(id).Data(do.GlassesUses{
			Status: status,
		}).Update()
		if err != nil {
			return err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return gerror.New("使用设备不存在，请联系管理员")
		}

		if status != 2 {
			_, err = dao.Glasses.Ctx(ctx).WherePri(glassesUses.GlassesId).Data(do.Glasses{
				Status: 1,
			}).Update()
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}
