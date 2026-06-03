package glasses

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

func (s *sGlasses) Create(ctx context.Context, req *model.GlassesCreateReq) (res *model.GlassesCreateRes, err error) {
	data := &do.Glasses{}
	if err = gconv.Scan(req, data); err != nil {
		return nil, gerror.Wrap(err, "参数转换失败")
	}

	data.Status = 1

	exists, err := dao.Glasses.Ctx(ctx).Where(do.Glasses{
		EquipmentSn: req.EquipmentSn,
	}).Exist()

	if err != nil {
		return nil, err
	}

	var id int64

	if exists {
		sqlResult, err := dao.Glasses.Ctx(ctx).Where(do.Glasses{
			EquipmentSn: req.EquipmentSn,
		}).Data(data).Update()

		if err != nil {
			return nil, gerror.Wrap(err, "更新失败[1]")
		}

		id, err = sqlResult.RowsAffected()

		if err != nil {
			return nil, gerror.Wrap(err, "更新失败[2]")
		}
	} else {
		id, err = dao.Glasses.Ctx(ctx).Data(data).InsertAndGetId()

		if err != nil {
			return nil, gerror.Wrap(err, "创建失败")
		}
	}

	res = &model.GlassesCreateRes{
		Id:        gconv.Uint64(id),
		CreatedAt: gtime.Now().Local(),
	}
	return res, err
}
