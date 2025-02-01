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

	if strings.Contains(chatreq.Model, "gpt") || strings.HasPrefix(chatreq.Model, "o1") || strings.HasPrefix(chatreq.Model, "o3") {
		openai.ChatProxy(c, &chatreq)
		return
	}

	if strings.HasPrefix(chatreq.Model, "claude") {
		claude.ChatProxy(c, &chatreq)
		return
	}

	if strings.HasPrefix(chatreq.Model, "gemini") || strings.HasPrefix(chatreq.Model, "learnlm") {
		google.ChatProxy(c, &chatreq)
		return
	}
}
