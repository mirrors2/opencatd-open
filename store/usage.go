package store

import (
	"log"
	"time"

	"github.com/Sakurasan/to"
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
	SKU             string    `gorm:"column:sku"`
	PromptUnits     int       `gorm:"column:prompt_units"`
	CompletionUnits int       `gorm:"column:completion_units"`
	TotalUnit       int       `gorm:"column:total_unit"`
	Cost            string    `gorm:"column:cost"`
	Date            time.Time `gorm:"column:date"`
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

type Tokens struct {
	UserID          int
	PromptCount     int
	CompletionCount int
	TotalTokens     int
	Cost            float64
	Model           string
	PromptHash      string
}

func Record(chatlog *Tokens) (err error) {
	u := &Usage{
		UserID:          chatlog.UserID,
		SKU:             chatlog.Model,
		PromptHash:      chatlog.PromptHash,
		PromptUnits:     chatlog.PromptCount,
		CompletionUnits: chatlog.CompletionCount,
		TotalUnit:       chatlog.TotalTokens,
		Cost:            to.String(chatlog.Cost),
		Date:            time.Now(),
	}
	err = usage.Create(u).Error
	return

}

func SumDaily(userid string) ([]Summary, error) {
	var count int64
	err := usage.Model(&DailyUsage{}).Where("user_id = ? and date = ?", userid, time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)).Count(&count).Error
	if err != nil {
		log.Println(err)
	}
	if count == 0 {

	} else {

	}

	return nil, nil
}

func insertSumDaily(uid string) error {
	nowstr := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
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

func updateSumDaily(uid string, date time.Time) error {
	var u = Usage{}
	err := usage.Exec(`SELECT 
	user_id,
	?,
	sku,
	SUM(prompt_units) AS prompt_units,
	SUM(completion_units) AS completion_units,
	SUM(total_unit) AS total_unit,
	SUM(cost) AS cost
	FROM usages 
	WHERE date >= ?
	AND user_id = ?`, date, date, uid).First(&u).Error
	if err != nil {
		return err
	}
	err = usage.Model(&DailyUsage{}).Where("user_id = ? and date = ?", uid, date).Updates(u).Error
	if err != nil {
		return err
	}
	return nil
}
