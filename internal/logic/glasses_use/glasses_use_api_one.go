package glasses_use

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (s *sGlassesUse) One(ctx context.Context, equipmentSn string) (res *model.GlassesUseOneRes, err error) {
	var glasses *model.Glasses
	var glassesUse *model.GlassesUse

	err = dao.Glasses.Ctx(ctx).Where(do.Glasses{
		EquipmentSn: equipmentSn,
	}).Scan(&glasses)

	if err != nil {
		return nil, gerror.Wrap(err, "获取设备失败")
	}

	if glasses == nil {
		return nil, gerror.New("设备不存在")
	}

	err = dao.GlassesUses.Ctx(ctx).Where(do.GlassesUses{
		GlassesId: glasses.Id,
		Status:    1,
	}).OrderDesc(dao.GlassesUses.Columns().Id).Limit(1).Scan(&glassesUse)

	if err != nil {
		return nil, gerror.Wrap(err, "查询设备使用记录失败")
	}

	if glassesUse == nil {
		return nil, gerror.New("设备使用不存在")
	}

	res = &model.GlassesUseOneRes{
		Glasses:    glasses,
		GlassesUse: glassesUse,
	}
	return
}
