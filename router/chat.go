package router

import (
	"net/http"
	"strings"

	"opencatd-open/pkg/claude"
	"opencatd-open/pkg/google"
	"opencatd-open/pkg/openai"

	"github.com/gin-gonic/gin"
)

func ChatHandler(c *gin.Context) {
	var chatreq openai.ChatCompletionRequest
	if err := c.ShouldBindJSON(&chatreq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.HasPrefix(chatreq.Model, "gpt") {
		openai.ChatProxy(c, &chatreq)
		return
	}

	if strings.HasPrefix(chatreq.Model, "claude") {
		claude.ChatProxy(c, &chatreq)
		return
	}

	if strings.HasPrefix(chatreq.Model, "gemini") {
		google.ChatProxy(c, &chatreq)
		return
	}
}
