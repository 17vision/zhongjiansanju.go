package glasses

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/model/entity"

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
	data.UseStatus = 1

	var entity *entity.Glasses
	err = dao.Glasses.Ctx(ctx).Where(dao.Glasses.Columns().EquipmentSn, req.EquipmentSn).Scan(&entity)
	if err != nil {
		return nil, gerror.Wrap(err, "查询设备失败")
	}

	if entity != nil {
		_, err = dao.Glasses.Ctx(ctx).
			Where(dao.Glasses.Columns().Id, entity.Id).
			OmitEmpty().
			Data(data).
			Update()

		if err != nil {
			return nil, gerror.Wrap(err, "更新失败")
		}

		res = &model.GlassesCreateRes{
			Id:        entity.Id,        // 返回真实的 ID
			CreatedAt: entity.CreatedAt, // 返回真实的创建时间
		}
	} else {
		// 3. 记录不存在 -> 插入
		id, err := dao.Glasses.Ctx(ctx).Data(data).InsertAndGetId()
		if err != nil {
			// 如果数据库有唯一索引，并发插入时报错可在此捕获，转为更新逻辑或直接报错
			return nil, gerror.Wrap(err, "创建失败")
		}

		res = &model.GlassesCreateRes{
			Id:        gconv.Uint64(id),
			CreatedAt: gtime.Now().Local(),
		}
	}
	return res, err
}
