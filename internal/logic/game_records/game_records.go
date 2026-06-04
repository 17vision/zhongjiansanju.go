package gamerecords

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

type sGameRecords struct{}

func init() {
	service.RegisterGameRecords(New())
}

func New() service.IGameRecords {
	return &sGameRecords{}
}

func (s *sGameRecords) Create(ctx context.Context, req model.GameRecordsCreateReq) (res *model.GameRecordsCreateRes, err error) {
	data := &do.GameRecords{}
	if err = gconv.Scan(req, data); err != nil {
		return nil, gerror.Wrap(err, "参数转换失败")
	}

	id, err := dao.GameRecords.Ctx(ctx).Data(data).InsertAndGetId()
	if err != nil {
		return nil, err
	}

	res = &model.GameRecordsCreateRes{
		Id:        gconv.Uint64(id),
		CreatedAt: gtime.Now().Local(),
	}
	return res, nil
}

func (s *sGameRecords) Update(ctx context.Context, req model.GameRecordsUpdateReq) (err error) {
	_, err = dao.GameRecords.Ctx(ctx).WherePri(req.Id).Data(do.GameRecords{
		EndAt: req.EndAt,
	}).Update()
	return
}
