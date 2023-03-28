package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"io/ioutil"
	"net/http"
	"path"
	"sort"
	"strconv"
	"time"
	"tool-attendance/log"
	"tool-attendance/model"
	"tool-attendance/utils/render"
)

type reqInitCalendar struct {
	Year int `uri:"year" binding:"required"`
}

func InitCalendar(c *gin.Context) {
	var req reqInitCalendar
	if err := c.ShouldBindUri(&req); err != nil {
		render.Json(c, render.ErrParams, err.Error())
		return
	}
	err := initCalendar(req.Year)
	if err != nil {
		render.Json(c, render.Failed, err.Error())
		return
	}
	render.Json(c, render.Ok, nil)
	return
}

func initCalendar(year int) error {
	list, err := getCalendar(year, 366)
	if err != nil {
		return err
	}
	dayList := make([]model.Calendar, 0, len(list))
	for _, v := range list {
		dayList = append(dayList, model.Calendar{
			ID:      0,
			Year:    v.Year,
			Month:   fmt.Sprintf("%d", v.Month),
			Date:    fmt.Sprintf("%d", v.Date),
			Week:    v.Week,
			Workday: v.Workday,
		})
	}
	err = model.MulCreateDate(dayList)
	if err != nil {
		return err
	}
	return nil
}

const (
	lateSymbol            = "1" // 迟到
	noLateSymbol          = ""  // 未迟到
	earlySymbol           = "1" // 早退
	noEarlySymbol         = ""  // 未早退
	cardSymbol            = "√" // 正常打卡
	noCardSymbol          = "×" // 未打卡
	unknownDurationSymbol = "-" // 未知的工作时长
)

type reqAttendanceDetail struct {
	Month int `uri:"month" binding:"required,gte=1,lte=12"`
}

// 统计备注：
// 出勤：工作日只要有打卡记录
// 旷工：工作日无打卡记录
// 迟到：工作日上班打卡在 9:30 后
// 早退：工作日下班打卡在 18:00 前
// 时长不足：工作日上下班打卡记录都有，但不足 9 小时
// 漏打卡：工作日只有上班卡，或只有下班卡

func AttendanceDetail(c *gin.Context) {
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

	tableRecords[1] = append(tableRecords[1], fmt.Sprintf("统计（本月出勤 %d 天）", needWorkDay))
	tableRecords[2] = append(tableRecords[2], []interface{}{"出勤", "旷工", "迟到", "早退", "时长不足", "漏打卡"}...)

	// 记录数据
	for i, userRecordList := range allRecordList {
		// 上班：
		userName := userRecordList[len(userRecordList)-1].Username
		if userName == "" {
			userName = userRecordList[len(userRecordList)-1].Firstname
		}
		onWorkRow := []interface{}{i + 1, userName, "上班"}
		// 下班
		offWorkRow := []interface{}{nil, nil, "下班"}
		// 时长
		durationRow := []interface{}{nil, nil, "时长"}
		// 迟到
		lateRow := []interface{}{nil, nil, "迟到"}
		// 早退
		earlyRow := []interface{}{nil, nil, "早退"}

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
				onWork   = noCardSymbol
				offWork  = noCardSymbol
				duration = unknownDurationSymbol
				late     = ""
				early    = ""
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
						onWork = record.OnworkTime.In(cstSh).Format(formatTime)
						if record.OnworkTime.Sub(onWorkLimitTime) <= 0 {
							late = noLateSymbol
						} else {
							// 迟到
							statLateDay++
							late = lateSymbol
						}
					} else {
						// 未打上班卡
						isLackCard = true
					}
					if !record.OffworkTime.IsZero() {
						// 打了下班卡
						offWork = record.OffworkTime.In(cstSh).Format(formatTime)
						if record.OffworkTime.Sub(offWorkLimitTime) >= 0 {
							early = noEarlySymbol
						} else {
							// 早退
							statEarlyDay++
							early = earlySymbol
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
						duration = fmt.Sprintf("%.1f", d)
					}
				} else {
					// 缺勤
					statAbsentDay++
					onWork = noCardSymbol
					offWork = noCardSymbol
					duration = ""
					late = ""
					early = ""
				}
			} else {
				// 休息日
				onWork = ""
				offWork = ""
				duration = ""
				late = ""
				early = ""
			}
			onWorkRow = append(onWorkRow, onWork)
			offWorkRow = append(offWorkRow, offWork)
			durationRow = append(durationRow, duration)
			lateRow = append(lateRow, late)
			earlyRow = append(earlyRow, early)
		}
		onWorkRow = append(onWorkRow, statWorkDay, statAbsentDay, statLateDay, statEarlyDay, statNotEnoughDurationDay, statLackCardDay)
		tableRecords = append(tableRecords, onWorkRow)
		tableRecords = append(tableRecords, offWorkRow)
		tableRecords = append(tableRecords, durationRow)
		tableRecords = append(tableRecords, lateRow)
		tableRecords = append(tableRecords, earlyRow)
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
	lastCel, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3+len(allRecordList)*5)
	_ = f.SetCellStyle(sheetName, "A1", lastCel, styleRecord)

	// 表头样式
	lastHeadCel, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3)
	_ = f.SetCellStyle(sheetName, "A2", lastHeadCel, styleHead)

	//设置列宽度
	//func (f *File) SetColWidth(sheet, startcol, endcol string, width float64) error
	//根据给定的工作表名称（大小写敏感）、列范围和宽度值设置单个或多个列的宽度。
	_ = f.SetColWidth(sheetName, "A", "A", 3)                             // 序号列
	_ = f.SetColWidth(sheetName, "B", "B", 7.5)                           // 姓名列
	_ = f.SetColWidth(sheetName, "C", "C", 5)                             // 日期-星期列
	_ = f.SetColWidth(sheetName, "D", calColumnTitle("D", totalDay-1), 8) // 数据列

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
		serialNumCel1, _ := excelize.JoinCellName("A", 3+1+i*4+i)
		serialNumCel2, _ := excelize.JoinCellName("A", 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, serialNumCel1, serialNumCel2)

		// 姓名
		nameCel1, _ := excelize.JoinCellName("B", 3+1+i*4+i)
		nameCel2, _ := excelize.JoinCellName("B", 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, nameCel1, nameCel2)

		// 出勤
		attCel1, _ := excelize.CoordinatesToCellName(3+totalDay+1, 3+1+i*4+i)

		attCel2, _ := excelize.CoordinatesToCellName(3+totalDay+1, 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, attCel1, attCel2)

		// 旷工
		absentCel1, _ := excelize.CoordinatesToCellName(3+totalDay+2, 3+1+i*4+i)
		absentCel2, _ := excelize.CoordinatesToCellName(3+totalDay+2, 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, absentCel1, absentCel2)

		// 迟到
		lateCel1, _ := excelize.CoordinatesToCellName(3+totalDay+3, 3+1+i*4+i)
		lateCel2, _ := excelize.CoordinatesToCellName(3+totalDay+3, 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, lateCel1, lateCel2)

		// 早退
		earlyCel1, _ := excelize.CoordinatesToCellName(3+totalDay+4, 3+1+i*4+i)
		earlyCel2, _ := excelize.CoordinatesToCellName(3+totalDay+4, 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, earlyCel1, earlyCel2)

		// 时长不足
		shortCel1, _ := excelize.CoordinatesToCellName(3+totalDay+5, 3+1+i*4+i)
		shortCel2, _ := excelize.CoordinatesToCellName(3+totalDay+5, 3+1+(i+1)*4+i)
		_ = f.MergeCell(sheetName, shortCel1, shortCel2)

		// 漏打卡
		missedCel1, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3+1+i*4+i)
		missedCel2, _ := excelize.CoordinatesToCellName(3+totalDay+6, 3+1+(i+1)*4+i)
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

	//// 调用 unoconv 工具将 Excel 文件转换为 HTML 文件
	//exec.Command("unoconv", "-f", "html", "-o", "cc.html", "cc.xlsx").Run()
	//
	//// 调用 wkhtmltoimage 工具将 HTML 文件转换为 PNG 图像
	//exec.Command("wkhtmltoimage", "--format", "png", "cc.html", "cc.png").Run()

	return
}

const (
	formatDayTime = "2006-01-02"
	formatTime    = "15:04:05"
)

const (
	cellStyleTitle = iota
	cellStyleHead
	cellStyleRecord
	cellStyleAbnormal
)

func getExcelStyle(f *excelize.File, style int) (int, error) {
	switch style {
	case cellStyleTitle:
		// 标题样式
		return f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 2},
				{Type: "right", Color: "000000", Style: 2},
				{Type: "top", Color: "000000", Style: 2},
				{Type: "bottom", Color: "000000", Style: 2},
			},
			Font: &excelize.Font{
				Bold:         true,
				Italic:       false,
				Underline:    "",
				Family:       "",
				Size:         18,
				Strike:       false,
				Color:        "",
				ColorIndexed: 0,
				ColorTheme:   nil,
				ColorTint:    0,
				VertAlign:    "",
			},
			Alignment: &excelize.Alignment{
				Horizontal:      "center", //水平居中
				Indent:          0,
				JustifyLastLine: false,
				ReadingOrder:    0,
				RelativeIndent:  0,
				ShrinkToFit:     false,
				TextRotation:    0,
				Vertical:        "center", //垂直居中
				WrapText:        false,
			},
		})

	case cellStyleHead:
		// 表头样式
		return f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 2},
				{Type: "right", Color: "000000", Style: 2},
				{Type: "top", Color: "000000", Style: 2},
				{Type: "bottom", Color: "000000", Style: 2},
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Pattern: 1,
				Color:   []string{"D1E9E9"},
				Shading: 0,
			},
			Font: &excelize.Font{
				Bold:         true,
				Italic:       false,
				Underline:    "",
				Family:       "",
				Size:         0,
				Strike:       false,
				Color:        "",
				ColorIndexed: 0,
				ColorTheme:   nil,
				ColorTint:    0,
				VertAlign:    "",
			},
			Alignment: &excelize.Alignment{
				Horizontal:      "center", //水平居中
				Indent:          0,
				JustifyLastLine: false,
				ReadingOrder:    0,
				RelativeIndent:  0,
				ShrinkToFit:     false,
				TextRotation:    0,
				Vertical:        "center", //垂直居中
				WrapText:        false,
			},
		})
	case cellStyleRecord:
		// 数据记录样式
		return f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 2},
				{Type: "right", Color: "000000", Style: 2},
				{Type: "top", Color: "000000", Style: 2},
				{Type: "bottom", Color: "000000", Style: 2},
			},
			Font: &excelize.Font{
				Bold:         false,
				Italic:       false,
				Underline:    "",
				Family:       "",
				Size:         0,
				Strike:       false,
				Color:        "",
				ColorIndexed: 0,
				ColorTheme:   nil,
				ColorTint:    0,
				VertAlign:    "",
			},
			Alignment: &excelize.Alignment{
				Horizontal:      "center", //水平居中
				Indent:          0,
				JustifyLastLine: false,
				ReadingOrder:    0,
				RelativeIndent:  0,
				ShrinkToFit:     false,
				TextRotation:    0,
				Vertical:        "center", //垂直居中
				WrapText:        false,
			},
		})
	case cellStyleAbnormal:
		return f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 2},
				{Type: "right", Color: "000000", Style: 2},
				{Type: "top", Color: "000000", Style: 2},
				{Type: "bottom", Color: "000000", Style: 2},
			},
			Font: &excelize.Font{
				Bold:         false,
				Italic:       false,
				Underline:    "",
				Family:       "",
				Size:         0,
				Strike:       false,
				Color:        "E60000",
				ColorIndexed: 0,
				ColorTheme:   nil,
				ColorTint:    0,
				VertAlign:    "",
			},
			Alignment: &excelize.Alignment{
				Horizontal:      "left", //水平居中
				Indent:          0,
				JustifyLastLine: false,
				ReadingOrder:    0,
				RelativeIndent:  0,
				ShrinkToFit:     false,
				TextRotation:    0,
				Vertical:        "center", //垂直居中
				WrapText:        false,
			},
		})
	default:
		return 0, errors.New("no find")
	}
}

var (
	weekday    = [7]int{7, 1, 2, 3, 4, 5, 6}
	weekChar   = []string{"", "一", "二", "三", "四", "五", "六", "日"}
	columnChar = []string{"", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
)

// 计算表格表头列
func calColumnTitle(s string, offset int) string {
	num, _ := getNum(s)
	return convertToTitle(num + offset)
}
func getNum(s string) (int, error) {
	for i, v := range columnChar {
		if v == s {
			return i, nil
		}
	}
	return 0, fmt.Errorf("invalid s")
}

func convertToTitle(columnNumber int) string {
	// 26个字母
	const cntLetter = 26

	// 结果
	var result string
	// 用于计算字母编码的切片
	var ch []int

	idx := columnNumber
	for idx > 0 {
		// 求余数
		tail := idx % cntLetter

		if tail == 0 {
			// 整除无余数，则用 26 来计算编码
			ch = append(ch, cntLetter)
			// 先减去26，再取整数部分进行下一次循环
			idx = (idx - cntLetter) / cntLetter
		} else {
			// 余数 用来计算编码
			ch = append(ch, tail)
			// 取整数部分进行下一次循环
			idx = idx / cntLetter
		}
	}

	// 循环切片，通过ASCII码计算出对应的字母后进行连接
	for _, v := range ch {
		result = string(v+65-1) + result
	}

	return result
}

// getWeek 根据指定日期获取星期
func getWeek(year, month, day int) string {
	var y, m, c int
	if month >= 3 {
		m = month
		y = year % 100
		c = year / 100
	} else {
		m = month + 12
		y = (year - 1) % 100
		c = (year - 1) / 100
	}
	week := y + (y / 4) + (c / 4) - 2*c + ((26 * (m + 1)) / 10) + day - 1
	if week < 0 {
		week = 7 - (-week)%7
	} else {
		week = week % 7
	}
	return weekChar[weekday[week]]
}

// getYearMonthToDay 查询指定年份指定月份有多少天
func getYearMonthToDay(year int, month int) int {
	// 有31天的月份
	day31 := map[int]struct{}{
		1:  struct{}{},
		3:  struct{}{},
		5:  struct{}{},
		7:  struct{}{},
		8:  struct{}{},
		10: struct{}{},
		12: struct{}{},
	}
	if _, ok := day31[month]; ok {
		return 31
	}
	// 有30天的月份
	day30 := map[int]struct{}{
		4:  struct{}{},
		6:  struct{}{},
		9:  struct{}{},
		11: struct{}{},
	}
	if _, ok := day30[month]; ok {
		return 30
	}
	// 计算是平年还是闰年
	if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
		// 得出2月的天数
		return 29
	}
	// 得出2月的天数
	return 28
}

// getFirstDateOfMonth 获取传入的时间所在月份的第一天，即某月第一天的0点
func getFirstDateOfMonth(d time.Time) time.Time {
	d = d.AddDate(0, 0, -d.Day()+1)
	return getZeroTime(d)
}

// getLastDateOfMonth 获取传入的时间所在月份的最后一天，即某月最后一天的24点
func getLastDateOfMonth(d time.Time) time.Time {
	return getFirstDateOfMonth(d).AddDate(0, 1, -1).Add(time.Hour*24 - time.Second*1)
}

// getZeroTime 获取某一天的0点时间
func getZeroTime(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}

type dayInfo struct {
	Year    int64 `json:"year"`
	Month   int64 `json:"month"`
	Date    int64 `json:"date"`
	Week    uint8 `json:"week"`
	Workday uint8 `json:"workday"`
}

type resCalendar struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List  []dayInfo `json:"list"`
		Page  int       `json:"page"`
		Size  int       `json:"size"`
		Total int       `json:"total"`
	} `json:"data"`
}

func getCalendar(year, size int) ([]dayInfo, error) {
	_url := fmt.Sprintf("https://api.apihubs.cn/holiday/get?year=%d&size=%d", year, size)
	req, _ := http.NewRequest("GET", _url, nil)
	//req.Header.Set("accept", "application/json")
	//req.Header.Set("X-API-KEY", nftScanAPIKey)
	srcResp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Log.Error("Get err:", err)
		return nil, err
	}
	body, _ := ioutil.ReadAll(srcResp.Body)
	resp := resCalendar{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Log.Error("Unmarshal err: body:", err, string(body))
		return nil, err
	}
	if resp.Code != 0 {
		log.Log.Error("fail.", resp.Msg)
		return nil, errors.New(resp.Msg)
	}

	return resp.Data.List, nil
}
