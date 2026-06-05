package glasses

import (
	"context"
	"fmt"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/pkg/utils"
)

func (s *sGlasses) Delete(ctx context.Context, req model.GlassesDeleteReq) (res *model.Result, err error) {
	if len(req.Ids) == 0 {
		res = &model.Result{Result: false, Message: "未指定删除项"}
		return
	}

	req.Ids = utils.Unique(req.Ids)

	usedIdsVal, err := dao.GlassesUses.Ctx(ctx).WhereIn(dao.GlassesUses.Columns().GlassesId, req.Ids).Distinct().Fields(dao.GlassesUses.Columns().GlassesId).Array()
	if err != nil {
		return
	}

	usedIds := make([]uint64, 0, len(usedIdsVal))
	for _, v := range usedIdsVal {
		usedIds = append(usedIds, v.Uint64())
	}

	deleteIds := utils.Difference(req.Ids, usedIds)

	if len(deleteIds) == 0 {
		res = &model.Result{
			Result:  false,
			Message: "所选记录均有使用记录，无法删除",
		}
		return
	}

	result, err := dao.Glasses.Ctx(ctx).
		WhereIn(dao.Glasses.Columns().Id, deleteIds).Delete()

	if err != nil {
		return
	}

	count, err := result.RowsAffected()
	if err != nil {
		return
	}

	var message string
	if len(usedIds) > 0 {
		message = fmt.Sprintf("有使用记录, 删除 %d 条记录", count)
	} else {
		message = fmt.Sprintf("全部删除, 删除 %d 条记录", len(req.Ids))
	}

	res = &model.Result{
		Result:  true,
		Message: message,
	}
	return
}
