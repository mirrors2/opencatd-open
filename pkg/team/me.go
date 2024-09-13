package team

import (
	"errors"
	"net/http"
	"opencatd-open/store"
	"time"

	"github.com/Sakurasan/to"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Handleinit(c *gin.Context) {
	user, err := store.GetUserByID(1)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			u := store.User{Name: "root", Token: uuid.NewString()}
			u.ID = 1
			if err := store.CreateUser(&u); err != nil {
				c.JSON(http.StatusForbidden, gin.H{
					"error": err.Error(),
				})
				return
			} else {
				rootToken = u.Token
				resJSON := User{
					false,
					int(u.ID),
					u.UpdatedAt.Format(time.RFC3339),
					u.Name,
					u.Token,
					u.CreatedAt.Format(time.RFC3339),
				}
				c.JSON(http.StatusOK, resJSON)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	if user.ID == uint(1) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "super user already exists, use cli to reset password",
		})
	}
}

func HandleMe(c *gin.Context) {
	token := c.GetHeader("Authorization")
	u, err := store.GetUserByToken(token[7:])
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	resJSON := User{
		false,
		int(u.ID),
		u.UpdatedAt.Format(time.RFC3339),
		u.Name,
		u.Token,
		u.CreatedAt.Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, resJSON)
}

func HandleMeUsage(c *gin.Context) {
	token := c.GetHeader("Authorization")
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
	user, err := store.GetUserByToken(token)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}
	usage, err := store.QueryUserUsage(to.String(user.ID), fromStr, toStr)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(200, usage)
}
