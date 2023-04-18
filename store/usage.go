package store

import (
	"time"
)

type DailyUsage struct {
	ID              int       `gorm:"column:id"`
	UserID          int       `gorm:"column:user_id primarykey"`
	Date            time.Time `gorm:"column:date"`
	SKU             string    `gorm:"column:sku"`
	PromptUnits     int       `gorm:"column:prompt_units"`
	CompletionUnits int       `gorm:"column:completion_units"`
	TotalUnit       int       `gorm:"column:total_unit"`
	Cost            string    `gorm:"column:cost"`
}

func (DailyUsage) TableName() string {
	return "daily_usages"
}

type Usage struct {
	ID              int       `gorm:"column:id"`
	PromptHash      string    `gorm:"column:prompt_hash"`
	UserID          int       `gorm:"column:user_id"`
	Date            time.Time `gorm:"column:date"`
	SKU             string    `gorm:"column:sku"`
	PromptUnits     int       `gorm:"column:prompt_units"`
	CompletionUnits int       `gorm:"column:completion_units"`
	TotalUnit       int       `gorm:"column:total_unit"`
	Cost            string    `gorm:"column:cost"`
}

func (Usage) TableName() string {
	return "usages"
}

type Summary struct {
	UserId int
	// SumPromptUnits     int
	// SumCompletionUnits int
	SumTotalUnit int
	SumCost      float64
}
type CalcUsage struct {
	UserID    int    `json:"userId,omitempty"`
	TotalUnit int    `json:"totalUnit,omitempty"`
	Cost      string `json:"cost,omitempty"`
}

func QueryUsage(from, to string) ([]CalcUsage, error) {
	var results = []CalcUsage{}
	err := usage.Model(&DailyUsage{}).Select(`user_id, 
	--SUM(prompt_units) AS prompt_units,
	-- SUM(completion_units) AS completion_units,
	SUM(total_unit) AS total_unit,
	SUM(cost) AS cost`).
		Group("user_id").
		Where("date >= ? AND date < ?", from, to).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func SumDaily(userid string) ([]Summary, error) {
	return nil, nil
}

func SumDailyV2(uid string) error {

	// err := usage.Model(&DailyUsage{}).
	// 	Select("user_id, '2023-04-18' as date, sku, SUM(prompt_units) as sum_prompt_units, SUM(completion_units) as sum_completion_units, SUM(total_unit) as sum_total_unit, SUM(cost) as sum_cost").
	// 	Where("date >= ?", "2023-04-18").
	// 	Where("user_id = ?", 2).
	// 	Create(&DailyUsage{}).Error
	nowstr := time.Now().Format("2006-01-02")
	err := usage.Exec(`INSERT INTO daily_usages
		(user_id, date, sku, prompt_units, completion_units, total_unit, cost)
		SELECT 
		user_id,
		?,
		sku,
		SUM(prompt_units) AS sum_prompt_units,
		SUM(completion_units) AS sum_completion_units,
		SUM(total_unit) AS sum_total_unit,
		SUM(cost) AS sum_cost
		FROM usages 
		WHERE date >= ?
		AND user_id = ?`, nowstr, nowstr, uid).Error
	if err != nil {
		return err
	}
	return nil
}
