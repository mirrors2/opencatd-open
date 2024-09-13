package error

import "github.com/gin-gonic/gin"

func ErrorData(message string) gin.H {
	return gin.H{
		"error": gin.H{
			"message": message,
		},
	}
}
