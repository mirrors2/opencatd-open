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

	// if chatreq.Messages[len(chatreq.Messages)-1].Role == "user" {
	// 	result, err := search.BingSearch(search.SearchParams{Query: string(chatreq.Messages[len(chatreq.Messages)-1].Content)})
	// 	if err == nil {
	// 		var msgs []openai.ChatCompletionMessage
	// 		for i, m := range chatreq.Messages {
	// 			var buf bytes.Buffer
	// 			buf.WriteString("根据我提问的语言回答我,我将提供一些从搜索引擎获取的信息(以websearch:开头)。你自行判断是否使用搜索引擎获取的内容。不要原封不动照抄,根据你自己的知识库提炼信息之后回答我\n\n")
	// 			if m.Role == "system" {
	// 				buf.Write(m.Content)
	// 				msgs = append(msgs, openai.ChatCompletionMessage{Role: m.Role, Content: buf.Bytes()})
	// 			} else {
	// 				msgs = append(msgs, openai.ChatCompletionMessage{Role: m.Role, Content: buf.Bytes()})
	// 			}
	// 			if i == len(chatreq.Messages)-1 {
	// 				m.Content = append(m.Content, json.RawMessage("\n\nwebsearch:")...)
	// 				m.Content = append(m.Content, json.RawMessage(result.(string))...)
	// 				msgs = append(msgs, openai.ChatCompletionMessage{Role: m.Role, Content: m.Content})
	// 			} else {
	// 				msgs = append(msgs, openai.ChatCompletionMessage{Role: m.Role, Content: m.Content})
	// 			}

	// 		}
	// 		chatreq.Messages = msgs
	// 	}
	// }

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
