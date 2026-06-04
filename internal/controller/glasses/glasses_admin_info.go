package glasses

import (
	"context"
	"time"

	"zjsj/api/glasses/admin"
	"zjsj/internal/model"
	"zjsj/internal/pkg/utils"
	"zjsj/internal/service"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

func (c *ControllerAdmin) Info(ctx context.Context, req *admin.InfoReq) (res *admin.InfoRes, err error) {
	listReq := model.GlassesListReq{
		PaginateReq: model.PaginateReq{
			Page:     1,
			PageSize: 100,
		},
	}

	data, err := service.Glasses().List(ctx, listReq)
	if err != nil {
		return
	}

	res = &admin.InfoRes{
		GlassesInfoRes: model.GlassesInfoRes{
			TotalCount:    len(data.Data),
			OnlineCount:   0,
			LowpowerCount: 0,
		},
	}

	var expiredIds = []int64{}
	referDate := gtime.Now().Add(-60 * time.Second)
	for _, item := range data.Data {
		if item.Status == 1 {
			res.OnlineCount++
		}

		// 统计低电量
		if item.BatteryLevel != "" {
			battery, err := utils.PercentageToFloat(item.BatteryLevel)
			if err == nil && battery < 0.2 {
				res.LowpowerCount++
			}
		}

		// 统计在线设备
		if item.UpdatedAt != nil && item.UpdatedAt.Before(referDate) {
			expiredIds = append(expiredIds, gconv.Int64(item.Id))
			if item.Status == 1 {
				res.OnlineCount--
			}
		}

		if len(expiredIds) > 0 {
			service.Glasses().UpdateStatus(ctx, expiredIds, 2)
		}
	}

	return
}
