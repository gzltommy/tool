package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"path"
	"sort"
	"strconv"
	"time"
	"tool-attendance/model"
	"tool-attendance/utils/render"
)

// 统计备注：
// 出勤：工作日只要有打卡记录
// 旷工：工作日无打卡记录
// 迟到：工作日上班打卡在 9:30 后
// 早退：工作日下班打卡在 18:00 前
// 时长不足：工作日上下班打卡记录都有，但不足 9 小时
// 漏打卡：工作日只有上班卡，或只有下班卡

func AttendanceRecord(c *gin.Context) {
	var req reqAttendanceDetail
	if err := c.ShouldBindUri(&req); err != nil {
		render.Json(c, render.ErrParams, err.Error())
		return
	}

	cstSh := time.FixedZone("CST", 8*3600)                                          // 东八
	yearS := c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().In(cstSh).Year())) // 年份
	year, _ := strconv.Atoi(yearS)

	// 获取打卡记录
	rt := time.Date(year, time.Month(req.Month), 1, 0, 0, 0, 0, cstSh)
	firstDate := getFirstDateOfMonth(rt)
	lastDate := getLastDateOfMonth(rt)
	recordList, err := model.FindRecordList(firstDate, lastDate)
	if err != nil {
		render.Json(c, render.Failed, err.Error())
		return
	}

	// 获取日历
	calendarMap, err := model.FindCalendarByMonth(int64(year), fmt.Sprintf("%d%02d", year, req.Month))
	if err != nil {
		render.Json(c, render.Failed, err.Error())
		return
	}
	if len(calendarMap) == 0 {
		// 初始化日历
		err = initCalendar(year)
		if err != nil {
			render.Json(c, render.Failed, err.Error())
			return
		}
		calendarMap, err = model.FindCalendarByMonth(int64(year), fmt.Sprintf("%d%02d", year, req.Month))
		if err != nil {
			render.Json(c, render.Failed, err.Error())
			return
		}
	}

	// 排序
	sort.Sort(model.RecordList(recordList))

	// 整理记录
	allRecordList := make([][]model.Record, 0, 50) // 内部的每个数组是单个用户的记录
	allRecordMap := make(map[string]int, 50)
	for _, v := range recordList {
		if index, ok := allRecordMap[v.UserId]; !ok {
			allRecordList = append(allRecordList, []model.Record{v})
			allRecordMap[v.UserId] = len(allRecordList) - 1
		} else {
			allRecordList[index] = append(allRecordList[index], v)
		}
	}

	f := excelize.NewFile()
	defer f.Close()

	// 计算该年份当月天数
	totalDay := getYearMonthToDay(year, req.Month)

	//--设置工作表名称
	//根据给定的新旧工作表名称（大小写敏感）重命名工作表。工作表名称最多允许使用 31 个字符，
	//此功能仅更改工作表的名称，而不会更新与单元格关联的公式或引用中的工作表名称。
	//因此使用此功能重命名工作表后可能导致公式错误或参考引用问题。
	sheetName := fmt.Sprintf("%d年%d月考勤记录", year, req.Month)
	_ = f.SetSheetName("Sheet1", sheetName) //设置工作表的名称

	tableRecords := [][]interface{}{
		{sheetName},        // 标题：2023年3月考勤记录
		{"序号", "姓名", "星期"}, // head：序号-姓名-星期
		{nil, nil, "日期"},   // head：日期
	}

	// 日期星期
	for i := 1; i <= totalDay; i++ {
		tableRecords[1] = append(tableRecords[1], getWeek(year, req.Month, i)) // 星期
		tableRecords[2] = append(tableRecords[2], i)                           // 日期
	}

	needWorkDay := 0
	for _, v := range calendarMap {
		if v.Workday == model.WorkDay {
			needWorkDay++
		}
	}

	tableRecords[1] = append(tableRecords[1], fmt.Sprintf("统计（本月需出勤 %d 天）", needWorkDay))
	tableRecords[2] = append(tableRecords[2], []interface{}{"出勤", "旷工", "迟到", "早退", "时长不足", "漏打卡"}...)

	// 记录数据
	for i, userRecordList := range allRecordList {
		// 上班：
		userName := userRecordList[len(userRecordList)-1].Firstname
		onWorkRow := []interface{}{i + 1, userName, "上班"}
		// 下班
		offWorkRow := []interface{}{nil, nil, "下班"}
		//// 时长
		//durationRow := []interface{}{nil, nil, "时长"}
		//// 迟到
		//lateRow := []interface{}{nil, nil, "迟到"}
		//// 早退
		//earlyRow := []interface{}{nil, nil, "早退"}

		// 用户打卡记录 map
		userRecordMap := make(map[string]model.Record, len(userRecordList))
		for _, v := range userRecordList {
			dayTimeStr := v.DaysDate.In(cstSh).Format(formatDayTime)
			userRecordMap[dayTimeStr] = v
		}

		// 数据记录
		var (
			statWorkDay              = 0 // 出勤天数（有一次打卡就算出勤）
			statAbsentDay            = 0 // 旷工天数（工作日一次打卡记录也没有）
			statLateDay              = 0 // 迟到天数
			statEarlyDay             = 0 // 早退天数
			statNotEnoughDurationDay = 0 // 时长不足天数
			statLackCardDay          = 0 // 漏打卡天数
		)
		for i := 1; i <= totalDay; i++ {
			var (
				onWork  = noCardSymbol
				offWork = noCardSymbol
				//duration = unknownDurationSymbol
				//late     = ""
				//early    = ""
			)
			// 工作日
			if calendarMap[fmt.Sprintf("%d%02d%02d", year, req.Month, i)].Workday == model.WorkDay {
				dayTimeStr := time.Date(year, time.Month(req.Month), i, 0, 0, 0, 0, cstSh).In(cstSh).Format(formatDayTime)
				onWorkLimitTime := time.Date(year, time.Month(req.Month), i, 9, 30, 0, 0, cstSh).In(cstSh)
				offWorkLimitTime := time.Date(year, time.Month(req.Month), i, 18, 0, 0, 0, cstSh).In(cstSh)
				if record, ok := userRecordMap[dayTimeStr]; ok {
					statWorkDay++
					isLackCard := false
					// 当日存在用户的打卡记录
					if !record.OnworkTime.IsZero() {
						// 打了上班卡
						//onWork = record.OnworkTime.In(cstSh).Format(formatTime)
						onWork = cardSymbol
						if record.OnworkTime.Sub(onWorkLimitTime) <= 0 {
							//late = noLateSymbol
						} else {
							// 迟到
							statLateDay++
							//late = lateSymbol
						}
					} else {
						// 未打上班卡
						isLackCard = true
					}
					if !record.OffworkTime.IsZero() {
						// 打了下班卡
						//offWork = record.OffworkTime.In(cstSh).Format(formatTime)
						offWork = cardSymbol
						if record.OffworkTime.Sub(offWorkLimitTime) >= 0 {
							//early = noEarlySymbol
						} else {
							// 早退
							statEarlyDay++
							//early = earlySymbol
						}
					} else {
						// 未打下班卡
						isLackCard = true
					}

					if isLackCard {
						statLackCardDay++
					}

					if !record.OnworkTime.IsZero() && !record.OffworkTime.IsZero() {
						d := float64(record.OffworkTime.Sub(record.OnworkTime)) / float64(time.Hour)
						if d < 9 {
							// 工作时长不足
							statNotEnoughDurationDay++

						}
						//duration = fmt.Sprintf("%.1f", d)
					}
				} else {
					// 缺勤
					statAbsentDay++
					onWork = noCardSymbol
					//offWork = noCardSymbol
					//duration = ""
					//late = ""
					//early = ""
				}
			} else {
				// 休息日
				onWork = ""
				offWork = ""
				//duration = ""
				//late = ""
				//early = ""
			}
			onWorkRow = append(onWorkRow, onWork)
			offWorkRow = append(offWorkRow, offWork)
			//durationRow = append(durationRow, duration)
			//lateRow = append(lateRow, late)
			//earlyRow = append(earlyRow, early)
		}
		onWorkRow = append(onWorkRow, statWorkDay, statAbsentDay, statLateDay, statEarlyDay, statNotEnoughDurationDay, statLackCardDay)
		tableRecords = append(tableRecords, onWorkRow)
		tableRecords = append(tableRecords, offWorkRow)
		//tableRecords = append(tableRecords, durationRow)
		//tableRecords = append(tableRecords, lateRow)
		//tableRecords = append(tableRecords, earlyRow)
	}

	for i, obj := range tableRecords {
		//--根据行和列拼接单元格名称
		name, _ := excelize.JoinCellName("A", i+1)

		//--按行赋值
		//根据给定的工作表名称（大小写敏感）、起始坐标和 slice 类型引用按行赋值。
		//例如，在名为 Sheet1 的工作簿第 6 行上，以 B6 单元格作为起始坐标按行赋值：
		//err := f.SetSheetRow("Sheet1", "B6", &[]interface{}{"1", nil, 2})
		_ = f.SetSheetRow(sheetName, name, &obj)
	}

	for i := 1; i <= totalDay; i++ {
		tableRecords[1] = append(tableRecords[1], getWeek(year, req.Month, i)) // 星期
		tableRecords[2] = append(tableRecords[2], i)                           // 日期
	}

	//--单元格样式
	//func (f *File) SetCellStyle(sheet, hcell, vcell string, styleID int) error
	//根据给定的工作表名、单元格坐标区域和样式索引设置单元格的值
	//。样式索引可以通过 NewStyle 函数获取。
	//注意，在同一个坐标区域内的 diagonalDown 和 diagonalUp 需要保持颜色一致。
	//SetCellStyle 将覆盖单元格的已有样式，而不会将样式与已有样式叠加或合并。
	styleTitle, _ := getExcelStyle(f, cellStyleTitle)       // 标题样式
	styleHead, _ := getExcelStyle(f, cellStyleHead)         // 表头样式
	styleRecord, _ := getExcelStyle(f, cellStyleRecord)     // 数据记录样式
	styleAbnormal, _ := getExcelStyle(f, cellStyleAbnormal) // 异常记录
	_ = styleAbnormal

	// 默认样式
	lastCel, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3+len(allRecordList)*2)
	_ = f.SetCellStyle(sheetName, "A1", lastCel, styleRecord)

	// 表头样式
	lastHeadCel, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3)
	_ = f.SetCellStyle(sheetName, "A2", lastHeadCel, styleHead)

	//设置列宽度
	//func (f *File) SetColWidth(sheet, startcol, endcol string, width float64) error
	//根据给定的工作表名称（大小写敏感）、列范围和宽度值设置单个或多个列的宽度。
	_ = f.SetColWidth(sheetName, "A", "A", 5)                             // 序号列
	_ = f.SetColWidth(sheetName, "B", "B", 10)                            // 姓名列
	_ = f.SetColWidth(sheetName, "C", "C", 5)                             // 日期-星期列
	_ = f.SetColWidth(sheetName, "D", calColumnTitle("D", totalDay-1), 4) // 数据列

	//--合并单元格
	//根据给定的工作表名（大小写敏感）和单元格坐标区域合并单元格。合并区域内仅保留左上角单元格的值，其他单元格的值将被忽略。
	//例如，合并名为 Sheet1 的工作表上 D3:E9 区域内的单元格：
	//err := f.MergeCell("Sheet1", "D3", "E9")
	//如果给定的单元格坐标区域与已有的其他合并单元格相重叠，已有的合并单元格将会被删除。

	// 标题
	titleCel, _ := excelize.CoordinatesToCellName(3+totalDay+6, 1)
	_ = f.SetCellStyle(sheetName, "A1", titleCel, styleTitle)
	_ = f.MergeCell(sheetName, "A1", titleCel)

	// heda-序号
	_ = f.MergeCell(sheetName, "A2", "A3")

	// heda-姓名
	_ = f.MergeCell(sheetName, "B2", "B3")

	// 统计
	statCel1, _ := excelize.CoordinatesToCellName(1+3+totalDay, 2)
	statCel2, _ := excelize.CoordinatesToCellName(1+3+totalDay+5, 2)
	_ = f.MergeCell(sheetName, statCel1, statCel2)

	// 记录
	for i, _ := range allRecordList {
		// 序号
		serialNumCel1, _ := excelize.JoinCellName("A", 3+1+i*1+i)
		serialNumCel2, _ := excelize.JoinCellName("A", 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, serialNumCel1, serialNumCel2)

		// 姓名
		nameCel1, _ := excelize.JoinCellName("B", 3+1+i*1+i)
		nameCel2, _ := excelize.JoinCellName("B", 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, nameCel1, nameCel2)

		// 出勤
		attCel1, _ := excelize.CoordinatesToCellName(3+totalDay+1, 3+1+i*1+i)
		attCel2, _ := excelize.CoordinatesToCellName(3+totalDay+1, 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, attCel1, attCel2)

		// 旷工
		absentCel1, _ := excelize.CoordinatesToCellName(3+totalDay+2, 3+1+i*1+i)
		absentCel2, _ := excelize.CoordinatesToCellName(3+totalDay+2, 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, absentCel1, absentCel2)

		// 迟到
		lateCel1, _ := excelize.CoordinatesToCellName(3+totalDay+3, 3+1+i*1+i)
		lateCel2, _ := excelize.CoordinatesToCellName(3+totalDay+3, 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, lateCel1, lateCel2)

		// 早退
		earlyCel1, _ := excelize.CoordinatesToCellName(3+totalDay+4, 3+1+i*1+i)
		earlyCel2, _ := excelize.CoordinatesToCellName(3+totalDay+4, 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, earlyCel1, earlyCel2)

		// 时长不足
		shortCel1, _ := excelize.CoordinatesToCellName(3+totalDay+5, 3+1+i*1+i)
		shortCel2, _ := excelize.CoordinatesToCellName(3+totalDay+5, 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, shortCel1, shortCel2)

		// 漏打卡
		missedCel1, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3+1+i*1+i)
		missedCel2, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3+1+(i+1)*1+i)
		_ = f.MergeCell(sheetName, missedCel1, missedCel2)
	}

	// 保存记录
	filePath := fmt.Sprintf("./attendance_record.xlsx")
	if err := f.SaveAs(filePath); err != nil {
		render.Json(c, render.Failed, err)
		return
	}

	fileName := path.Base(filePath)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Transfer-Encoding", "binary")
	c.File(filePath)
	return
}
