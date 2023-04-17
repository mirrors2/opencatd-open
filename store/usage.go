package store

import "time"

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

func QueryUsage(from, to time.Time) error {
	return nil
}
