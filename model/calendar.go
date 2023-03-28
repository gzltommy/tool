package model

const (
	WorkDay = 1
	RestDay = 2
)

type Calendar struct {
	ID      int64  `gorm:"column:id" json:"id"`
	Year    int64  `gorm:"column:year" json:"year"`       // 2023
	Month   string `gorm:"column:month" json:"month"`     // 202305
	Date    string `gorm:"column:date" json:"date"`       // 20230504
	Week    uint8  `gorm:"column:week" json:"week"`       //  星期几
	Workday uint8  `gorm:"column:workday" json:"workday"` //  1:工作日；2：非工作日
}

func MulCreateDate(list []Calendar) error {
	return db.Model(&Calendar{}).CreateInBatches(list, 100).Error
}

func FindCalendarByMonth(year int64, month string) (map[string]Calendar, error) {
	var rows []Calendar
	err := db.Model(&Calendar{}).Where("year=? and month=?", year, month).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	resMap := make(map[string]Calendar, len(rows))
	for _, v := range rows {
		resMap[v.Date] = v
	}
	return resMap, nil
}
