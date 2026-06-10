package gamerecords

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"zjsj/internal/dao"
	"zjsj/internal/model"
	"zjsj/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/xuri/excelize/v2"
)

func (s *sGameRecords) Export(ctx context.Context, req model.GameRecordExportReq) (res model.GameRecordExportRes, err error) {

	// 先查询所有的眼镜设备数据

	columns := dao.GameRecords.Columns()

	// 构建查询条件
	query := dao.GameRecords.Ctx(ctx)
	if req.StartAt != nil && req.EndAt != nil {
		query = query.WhereGT(columns.CreatedAt, req.StartAt).WhereLT(columns.CreatedAt, req.EndAt)
	}

	// 获取HTTP请求对象，设置响应头
	r := g.RequestFromCtx(ctx)
	filename := fmt.Sprintf("体验列表_%s.xlsx", gtime.Now().Format("YmdHis"))
	encodedFilename := url.QueryEscape(filename)
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", encodedFilename, encodedFilename))
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.WriteHeader(http.StatusOK)

	// 创建Excel文件
	f := excelize.NewFile()
	sheetName := "体验数据"
	f.SetSheetName("Sheet1", sheetName)

	// 创建流式写入器
	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		g.Log().Error(ctx, "创建流式写入器失败", err)
		return res, fmt.Errorf("创建导出文件失败")
	}

	// =============================================
	// ✅ 新增：设置每一列的固定宽度（流式导出最佳方案）
	// =============================================
	// 列顺序：A B C D E F G H I
	// 对应表头：ID 设备名称 设备Sn码 用户昵称 体验模型 连接时间 开始时间 结束时间 体验时长
	colWidths := []float64{
		8,  // A列：ID
		20, // B列：设备名称
		25, // C列：设备Sn码
		15, // D列：用户昵称
		15, // E列：体验模型
		20, // F列：连接时间
		20, // G列：开始时间
		20, // H列：结束时间
		12, // I列：体验时长
	}

	for i, width := range colWidths {
		colName, _ := excelize.ColumnNumberToName(i + 1) // i从0开始，列从1开始
		f.SetColWidth(sheetName, colName, colName, width)
	}

	// 资源释放
	defer func() {
		if err := sw.Flush(); err != nil {
			g.Log().Error(ctx, "刷新流式写入器失败", err)
		}
		if err := f.Write(r.Response.Writer); err != nil {
			g.Log().Error(ctx, "写入响应失败", err)
		}
		if err := f.Close(); err != nil {
			g.Log().Error(ctx, "关闭Excel文件失败", err)
		}
	}()

	// 写入表头
	headers := []interface{}{
		"ID", "设备名称", "设备Sn码", "用户昵称", "体验模型",
		"连接时间", "开始时间", "结束时间", "体验时长",
	}
	if err = sw.SetRow("A1", headers); err != nil {
		g.Log().Error(ctx, "写入表头失败", err)
		return res, fmt.Errorf("写入表头失败")
	}

	// 先把设备取出来
	var glasses []*entity.Glasses
	if err = dao.Glasses.Ctx(ctx).Scan(&glasses); err != nil {
		g.Log().Error(ctx, "获取眼镜列表失败", err)
		return res, fmt.Errorf("获取眼镜列表失败")
	}

	type SimpleGlasses struct {
		Id          uint64
		Name        string
		EquipmentSn string
	}

	var glassesMap = make(map[uint64]*SimpleGlasses)
	for _, glasses := range glasses {
		glassesMap[glasses.Id] = &SimpleGlasses{
			Id:          glasses.Id,
			Name:        glasses.Name,
			EquipmentSn: glasses.EquipmentSn,
		}
	}

	// 流式分批查询并写入数据
	rowNum := 2
	chunkSize := 1000
	var chunkErr error

	// Chunk 里边 wit 无用
	query.Chunk(chunkSize, func(result gdb.Result, err error) bool {
		if err != nil {
			chunkErr = err
			return false
		}

		if chunkErr != nil {
			return false
		}

		var records []*model.GameRecord
		if err := gconv.Structs(result, &records); err != nil {
			g.Log().Error(ctx, "数据转换失败", err)
			chunkErr = err
			return false
		}

		var glassesUseIds []uint64
		for _, record := range records {
			glassesUseIds = append(glassesUseIds, record.GlassesUseId)
		}

		var glassesUses []*entity.GlassesUses
		var glassesUseMap = make(map[uint64]*entity.GlassesUses)
		if len(glassesUseIds) > 0 {
			err = dao.GlassesUses.Ctx(ctx).
				WhereIn(dao.GlassesUses.Columns().Id, glassesUseIds).
				Scan(&glassesUses)
			if err != nil {
				g.Log().Error(ctx, "查询GlassesUse失败", err)
				chunkErr = err
				return false
			}
		}

		for _, glassesUse := range glassesUses {
			glassesUseMap[glassesUse.Id] = glassesUse
		}

		for _, record := range records {
			g.Log().Info(ctx, "处理第", rowNum-1, "条数据，ID：", record.Id)

			var (
				glassesName  = ""
				equipmentSn  = ""
				nickname     = ""
				modelName    = ""
				connectAtStr = ""
				startAtStr   = ""
				endAtStr     = ""
				duration     = "未知"
			)

			if glassesUse, ok := glassesUseMap[record.GlassesUseId]; ok {
				nickname = glassesUse.Nickname
				modelName = glassesUse.Model
				if glasses, ok := glassesMap[glassesUse.GlassesId]; ok {
					glassesName = glasses.Name
					equipmentSn = glasses.EquipmentSn
				}
			}

			// 2. 时间字段保护
			if record.ConnectAt != nil {
				connectAtStr = record.ConnectAt.Format("Y-m-d H:i:s")
			}
			if record.StartAt != nil {
				startAtStr = record.StartAt.Format("Y-m-d H:i:s")
			}
			if record.EndAt != nil {
				endAtStr = record.EndAt.Format("Y-m-d H:i:s")
			}

			// 3. 时长计算保护
			if record.StartAt != nil && record.EndAt != nil {
				diff := int(record.EndAt.Sub(record.StartAt).Seconds())
				if diff < 60 {
					duration = fmt.Sprintf("%d秒", diff)
				} else {
					minutes := int(math.Floor(float64(diff / 60)))
					seconds := diff % 60
					duration = fmt.Sprintf("%d分%d秒", minutes, seconds)
				}
			}

			// ✅ 现在绝对不会panic了
			row := []interface{}{
				record.Id,
				glassesName,
				equipmentSn,
				nickname,
				modelName,
				connectAtStr,
				startAtStr,
				endAtStr,
				duration,
			}

			// g.Log().Info(ctx, "写入数据:", gjson.MustEncodeString(row))

			cell, _ := excelize.CoordinatesToCellName(1, rowNum)
			if err := sw.SetRow(cell, row); err != nil {
				g.Log().Error(ctx, "写入行失败", err)
				chunkErr = err
				return false
			}

			rowNum++
		}

		return true
	})

	if chunkErr != nil {
		g.Log().Error(ctx, "导出数据失败", chunkErr)
		return res, fmt.Errorf("导出数据失败")
	}

	// 返回导出总条数
	res.Total = rowNum - 2
	g.Log().Info(ctx, "导出成功，共导出", res.Total, "条数据")

	// 注意：这里不能调用 r.Response.WriteJsonExit()
	// 因为我们已经直接写入了文件流
	return res, nil
}
