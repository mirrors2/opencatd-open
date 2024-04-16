// https://docs.anthropic.com/claude/reference/messages_post

package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"opencatd-open/pkg/openai"
	"opencatd-open/pkg/tokenizer"
	"opencatd-open/store"
	"strings"

	"github.com/gin-gonic/gin"
)

func ChatProxy(c *gin.Context, chatReq *openai.ChatCompletionRequest) {
	ChatMessages(c, chatReq)
}

func ChatTextCompletions(c *gin.Context, chatReq *openai.ChatCompletionRequest) {

}

type ClaudeRequest struct {
	Model       string  `json:"model,omitempty"`
	Messages    any     `json:"messages,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
	System      string  `json:"system,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

func (c *ClaudeRequest) ByteJson() []byte {
	bytejson, _ := json.Marshal(c)
	return bytejson
}

type ClaudeMessages struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type VisionMessages struct {
	Role    string          `json:"role,omitempty"`
	Content []ClaudeContent `json:"content,omitempty"`
}

type ClaudeContent struct {
	Type   string        `json:"type,omitempty"`
	Source *ClaudeSource `json:"source,omitempty"`
	Text   string        `json:"text,omitempty"`
}

type ClaudeSource struct {
	Type      string `json:"type,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

type ClaudeResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	StopSequence any    `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

type ClaudeStreamResponse struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block"`
	Delta struct {
		Type         string `json:"type"`
		Text         string `json:"text"`
		StopReason   string `json:"stop_reason"`
		StopSequence any    `json:"stop_sequence"`
	} `json:"delta"`
	Message struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Content      []any  `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence any    `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func ChatMessages(c *gin.Context, chatReq *openai.ChatCompletionRequest) {
	// var haveImages bool

	usagelog := store.Tokens{Model: chatReq.Model}
	var claudReq ClaudeRequest
	claudReq.Model = chatReq.Model
	claudReq.Stream = chatReq.Stream
	claudReq.Temperature = chatReq.Temperature
	claudReq.TopP = chatReq.TopP
	claudReq.MaxTokens = 4096

	var msgs []any
	var prompt string
	for _, msg := range chatReq.Messages {
		if msg.Role == "system" {
			claudReq.System = string(msg.Content)
			continue
		}

		var visioncontent []openai.VisionContent
		if err := json.Unmarshal(msg.Content, &visioncontent); err != nil {
			prompt += "<" + msg.Role + ">: " + string(msg.Content) + "\n"

			var claudemsgs ClaudeMessages
			claudemsgs.Role = msg.Role
			claudemsgs.Content = string(msg.Content)
			msgs = append(msgs, claudemsgs)

		} else {
			if len(visioncontent) > 0 {
				var visionMessage VisionMessages
				visionMessage.Role = msg.Role

				for _, content := range visioncontent {
					var claudecontent []ClaudeContent
					if content.Type == "text" {
						prompt += "<" + msg.Role + ">: " + content.Text + "\n"
						claudecontent = append(claudecontent, ClaudeContent{Type: "text", Text: content.Text})
					} else if content.Type == "image_url" {
						if strings.HasPrefix(content.ImageURL.URL, "http") {
							fmt.Println("链接:", content.ImageURL.URL)
						} else if strings.HasPrefix(content.ImageURL.URL, "data:image") {
							fmt.Println("base64:", content.ImageURL.URL[:20])
						}
						// todo image tokens
						var mediaType string
						if strings.HasPrefix(content.ImageURL.URL, "data:image/jpeg") {
							mediaType = "image/jpeg"
						}
						if strings.HasPrefix(content.ImageURL.URL, "data:image/png") {
							mediaType = "image/png"
						}
						claudecontent = append(claudecontent, ClaudeContent{Type: "image", Source: &ClaudeSource{Type: "base64", MediaType: mediaType, Data: strings.Split(content.ImageURL.URL, ",")[1]}})
						// haveImages = true

					}
					visionMessage.Content = claudecontent
				}
				msgs = append(msgs, visionMessage)
			}
		}
		claudReq.Messages = msgs

		// if len(chatReq.Tools) > 0 {
		// 	tooljson, _ := json.Marshal(chatReq.Tools)
		// 	prompt += "<tools>: " + string(tooljson) + "\n"
		// }
	}

	usagelog.PromptCount = tokenizer.NumTokensFromStr(prompt, chatReq.Model)

	req, _ := http.NewRequest("POST", MessageEndpoint, strings.NewReader(fmt.Sprintf("%v", bytes.NewReader(claudReq.ByteJson()))))
	client := http.DefaultClient
	rsp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		io.Copy(c.Writer, rsp.Body)
		return
	}

	teeReader := io.TeeReader(rsp.Body, c.Writer)

	dataChan := make(chan string, 1)
	// stopChan := make(chan bool)

	var result string

	scanner := bufio.NewScanner(teeReader)

	go func() {
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 && bytes.HasPrefix(line, []byte("data: ")) {
				if bytes.HasPrefix(line, []byte("data: [DONE]")) {
					dataChan <- string(line) + "\n"
					break
				}
				var claudeResp ClaudeStreamResponse
				line = bytes.Replace(line, []byte("data: "), []byte(""), -1)
				line = bytes.TrimSpace(line)
				if err := json.Unmarshal(line, &claudeResp); err != nil {
					continue
				}

				if claudeResp.Type == "message_start" {
					if claudeResp.Message.Role != "" {
						result += "<" + claudeResp.Message.Role + ">"
					}
				} else if claudeResp.Type == "message_stop" {
					break
				}

				if claudeResp.Delta.Text != "" {
					result += claudeResp.Delta.Text
				}
				var choice openai.Choice
				choice.Delta.Role = claudeResp.Message.Role
				choice.Delta.Content = claudeResp.Delta.Text
				choice.FinishReason = claudeResp.Delta.StopReason

				chatResp := openai.ChatCompletionStreamResponse{
					Model:   chatReq.Model,
					Choices: []openai.Choice{choice},
				}
				dataChan <- "data: " + string(chatResp.ByteJson()) + "\n"
				if claudeResp.Delta.StopReason != "" {
					dataChan <- "\ndata: [DONE]\n"
				}
			} else {
				if !bytes.HasPrefix(line, []byte("event:")) {
					dataChan <- string(line) + "\n"
				}
			}
		}
		defer close(dataChan)
	}()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		if data, ok := <-dataChan; ok {
			if strings.HasPrefix(data, "data: ") {
				c.Writer.WriteString(data)
				// c.Writer.WriteString("\n\n")
			} else {
				c.Writer.WriteHeader(http.StatusBadGateway)
				c.Writer.WriteString(data)
			}
			c.Writer.Flush()
			return true
		}
		go func() {
			usagelog.CompletionCount = tokenizer.NumTokensFromStr(result, chatReq.Model)
			usagelog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(usagelog.Model, usagelog.PromptCount, usagelog.CompletionCount))
			if err := store.Record(&usagelog); err != nil {
				log.Println(err)
			}
			if err := store.SumDaily(usagelog.UserID); err != nil {
				log.Println(err)
			}
		}()
		return false
	})
}
