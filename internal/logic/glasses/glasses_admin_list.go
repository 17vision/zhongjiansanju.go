package glasses

import (
	"context"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/do"
)

func (s *sGlasses) List(ctx context.Context, req model.GlassesListReq) (res *model.GlassesListRes, err error) {
	columns := dao.Glasses.Columns()

	query := dao.Glasses.Ctx(ctx).OrderDesc(columns.Id)
	if req.UseStatus != 0 {
		query = query.Where(do.Glasses{
			UseStatus: req.UseStatus,
		})
	}

	if req.Name != "" {
		query = query.WhereLike(columns.Name, "%"+req.Name+"%")
	}

	var data []*model.Glasses
	var total int
	if err = query.Page(req.Page, req.PageSize).ScanAndCount(&data, &total, true); err != nil {
		return nil, err
	}

	statusArr := []string{"", "在线", "离线"}
	for _, item := range data {
		item.StatusStr = statusArr[item.Status]
	}

	res = &model.GlassesListRes{}
	res.Total = total
	res.Page = req.Page
	res.PageSize = req.PageSize
	res.Data = data
	return
}
