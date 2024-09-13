package team

import (
	"log"
	"net/http"
	"opencatd-open/store"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	rootToken string
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rootToken == "" {
			u, err := store.GetUserByID(uint(1))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			rootToken = u.Token
		}
		token := c.GetHeader("Authorization")
		if token == "" || token[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		if store.IsExistAuthCache(token[7:]) {
			if strings.HasPrefix(c.Request.URL.Path, "/1/me") {
				c.Next()
				return
			}
		}
		if token[7:] != rootToken {
			u, err := store.GetUserByID(uint(1))
			if err != nil {
				log.Println(err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			if token[:7] != u.Token {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			rootToken = u.Token
			store.LoadAuthCache()
		}
		// 可以在这里对 token 进行验证并检查权限

		c.Next()
	}
}
