package model

import "time"

type Record struct {
	UserId      string    `gorm:"column:user_id" json:"user_id"`
	Firstname   string    `gorm:"column:firstname" json:"firstname"`
	Username    string    `gorm:"column:username" json:"username"`
	DaysDate    time.Time `gorm:"column:days_date" json:"days_date"`
	OnworkTime  time.Time `gorm:"column:onwork_time" json:"onwork_time"`
	OffworkTime time.Time `gorm:"column:offwork_time" json:"offwork_time"`
}

func FindRecordList(beginDay, endDay time.Time) ([]Record, error) {
	var raws []Record
	err := db.Model(&Record{}).
		Where("? <= days_date and days_date <= ?", beginDay, endDay).
		//Limit(4).
		//Order("user_id").
		Find(&raws).Error
	return raws, err
}

type RecordList []Record

func (l RecordList) Len() int {
	return len(l)
}

func (l RecordList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l RecordList) Less(i, j int) bool {
	if l[i].UserId < l[j].UserId {
		return true
	} else if l[i].UserId > l[j].UserId {
		return false
	} else {
		if l[i].DaysDate.Sub(l[j].DaysDate) < 0 {
			return true
		} else {
			return false
		}
	}
}
