package team

import (
	"net/http"
	"opencatd-open/store"
	"time"

	"github.com/gin-gonic/gin"
)

func HandleUsage(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	getMonthStartAndEnd := func() (start, end string) {
		loc, _ := time.LoadLocation("Local")
		now := time.Now().In(loc)

		year, month, _ := now.Date()

		startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		start = startOfMonth.Format("2006-01-02")
		end = endOfMonth.Format("2006-01-02")
		return
	}
	if fromStr == "" || toStr == "" {
		fromStr, toStr = getMonthStartAndEnd()
	}

	usage, err := store.QueryUsage(fromStr, toStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, usage)
}
