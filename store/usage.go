package store

import (
	"errors"
	"time"

	"github.com/Sakurasan/to"
	"gorm.io/gorm"
)

type DailyUsage struct {
	ID              int       `gorm:"column:id"`
	UserID          int       `gorm:"column:user_id";primaryKey`
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
	UserId             int     `gorm:"column:user_id"`
	SumPromptUnits     int     `gorm:"column:sum_prompt_units"`
	SumCompletionUnits int     `gorm:"column:sum_completion_units"`
	SumTotalUnit       int     `gorm:"column:sum_total_unit"`
	SumCost            float64 `gorm:"column:sum_cost"`
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
	printf('%.6f', SUM(cost)) AS cost`).
		Group("user_id").
		Where("date >= ? AND date < ?", from, to).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func QueryUserUsage(userid, from, to string) (*CalcUsage, error) {
	var results = new(CalcUsage)
	err := usage.Model(&DailyUsage{}).Select(`user_id, 
	--SUM(prompt_units) AS prompt_units,
	-- SUM(completion_units) AS completion_units,
	SUM(total_unit) AS total_unit,
	printf('%.6f', SUM(cost)) AS cost`).
		Where("user_id = ? AND date >= ? AND date < ?", userid, from, to).
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
	Cost            string
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

func SumDaily(userid int) error {
	var count int64
	err := usage.Model(&DailyUsage{}).Where("user_id = ? and date = ?", userid, time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)).Count(&count).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if count == 0 {
		if err := insertSumDaily(userid); err != nil {
			return err
		}
	} else {
		if err := updateSumDaily(userid, time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)); err != nil {
			return err
		}
	}
	return nil
}

func insertSumDaily(uid int) error {
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

func updateSumDaily(uid int, date time.Time) error {
	// var u = Summary{}
	err := usage.Model(&Usage{}).Exec(`UPDATE daily_usages
	SET 
	prompt_units = (SELECT SUM(prompt_units) FROM usages WHERE user_id = daily_usages.user_id AND date >= daily_usages.date),
	completion_units = (SELECT SUM(completion_units) FROM usages WHERE user_id = daily_usages.user_id AND date >= daily_usages.date),
	total_unit = (SELECT SUM(total_unit) FROM usages WHERE user_id = daily_usages.user_id AND date >= daily_usages.date),
	cost = (SELECT SUM(cost) FROM usages WHERE user_id = daily_usages.user_id AND date >= daily_usages.date)
	WHERE user_id = ? AND date >= ?`, uid, date).Error
	if err != nil {
		return err
	}
	return nil
}
