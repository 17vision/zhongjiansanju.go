package glasses_use

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

func (s *sGlassesUse) Create(ctx context.Context, req *model.GlassesUseCreateReq) (res *model.GlassesUseCreateRes, err error) {
	data := &do.GlassesUses{}
	if err = gconv.Scan(req, data); err != nil {
		return nil, gerror.Wrap(err, "数据转换失败")
	}
	data.Status = 1

	var id int64
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		var glassesUse *entity.GlassesUses
		err := dao.GlassesUses.Ctx(ctx).
			Where(do.GlassesUses{
				GlassesId: req.GlassesId,
				Status:    1,
			}).
			OrderDesc(dao.GlassesUses.Columns().Id).
			LockUpdate(). // 排他悲观锁
			Scan(&glassesUse)
		if err != nil {
			return gerror.Wrap(err, "查询在用记录失败")
		}

		var innerErr error
		if glassesUse != nil {
			resUpdate, innerErr := dao.GlassesUses.Ctx(ctx).
				WherePri(glassesUse.Id).
				Data(data).
				Update()
			if innerErr != nil {
				return gerror.Wrap(innerErr, "更新在用记录失败")
			}
			affected, innerErr := resUpdate.RowsAffected()
			if innerErr != nil {
				return innerErr
			}
			if affected == 0 {
				return gerror.New("在用记录已不存在")
			}

			id = int64(glassesUse.Id)
		} else {
			id, innerErr = dao.GlassesUses.Ctx(ctx).Data(data).InsertAndGetId()
			if innerErr != nil {
				return gerror.Wrap(innerErr, "新增在用记录失败")
			}
		}

		// 同事务内更新眼镜状态，不要调用外层service
		_, innerErr = dao.Glasses.Ctx(ctx).
			WherePri(req.GlassesId).
			Data(do.Glasses{Status: 2}).
			Update()
		if innerErr != nil {
			return gerror.Wrap(innerErr, "更新眼镜状态失败")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	res = &model.GlassesUseCreateRes{
		Id: gconv.Uint64(id),
	}
	return res, nil
}
