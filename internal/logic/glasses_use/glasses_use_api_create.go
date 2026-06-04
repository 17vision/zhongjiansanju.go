package glasses_use

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/model/entity"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

func (s *sGlassesUse) Create(ctx context.Context, req *model.GlassesUseCreateReq) (res *model.GlassesUseCreateRes, err error) {
	data := &do.GlassesUses{}
	if err = gconv.Scan(req, data); err != nil {
		return nil, gerror.Wrap(err, "数据转换失败")
	}
	data.Status = 1

	columns := dao.GlassesUses.Columns()

	var glassesUse *entity.GlassesUses
	err = dao.GlassesUses.Ctx(ctx).Where(do.GlassesUses{
		GlassesId: req.GlassesId,
		Status:    1,
	}).OrderDesc(columns.Id).Scan(&glassesUse)

	if err != nil {
		return nil, gerror.Wrap(err, "获取数据失败")
	}

	var id int64
	if glassesUse != nil {
		_, err = dao.GlassesUses.Ctx(ctx).Where(do.GlassesUses{
			Id: glassesUse.Id,
		}).Data(data).Update()

		id = int64(glassesUse.Id)
	} else {
		id, err = dao.GlassesUses.Ctx(ctx).Data(data).InsertAndGetId()
	}

	if err != nil {
		return nil, gerror.Wrap(err, "提交数据失败")
	}

	// 将设备状置为 2
	service.Glasses().UpdateUseStatus(ctx, []int64{req.GlassesId}, 2)

	res = &model.GlassesUseCreateRes{
		Id:        gconv.Uint64(id),
		CreatedAt: gtime.Now().Local(),
	}
	return
}
