package team

import (
	"net/http"
	"opencatd-open/store"

	"github.com/Sakurasan/to"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type User struct {
	IsDelete  bool   `json:"IsDelete,omitempty"`
	ID        int    `json:"id,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Name      string `json:"name,omitempty"`
	Token     string `json:"token,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func HandleUsers(c *gin.Context) {
	users, err := store.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, users)
}

func HandleAddUser(c *gin.Context) {
	var body User
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	if len(body.Name) == 0 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid user name"})
		return
	}

	if err := store.AddUser(body.Name, uuid.NewString()); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	u, err := store.GetUserByName(body.Name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func HandleDelUser(c *gin.Context) {
	id := to.Int(c.Param("id"))
	if id <= 1 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid user id"})
		return
	}
	if err := store.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func HandleResetUserToken(c *gin.Context) {
	id := to.Int(c.Param("id"))
	newtoken := c.Query("token")
	if newtoken == "" {
		newtoken = uuid.NewString()
	}

	if err := store.UpdateUser(uint(id), newtoken); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	u, err := store.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if u.ID == uint(1) {
		rootToken = u.Token
	}
	c.JSON(http.StatusOK, u)
}
