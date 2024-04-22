package google

import (
	"context"
	"encoding/base64"
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
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GeminiChatRequest struct {
	Contents []GeminiContent `json:"contents,omitempty"`
}

func (g GeminiChatRequest) ByteJson() []byte {
	bytejson, _ := json.Marshal(g)
	return bytejson
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts,omitempty"`
}
type GeminiPart struct {
	Text string `json:"text,omitempty"`
	// InlineData GeminiPartInlineData `json:"inlineData,omitempty"`
}
type GeminiPartInlineData struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"` // base64
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		Index         int    `json:"index"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	PromptFeedback struct {
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"promptFeedback"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type            string `json:"@type"`
			FieldViolations []struct {
				Field       string `json:"field"`
				Description string `json:"description"`
			} `json:"fieldViolations"`
		} `json:"details"`
	} `json:"error"`
}

func ChatProxy(c *gin.Context, chatReq *openai.ChatCompletionRequest) {
	usagelog := store.Tokens{Model: chatReq.Model}

	token, _ := c.Get("localuser")

	lu, err := store.GetUserByToken(token.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}
	usagelog.UserID = int(lu.ID)
	var prompts []genai.Part
	var prompt string
	for _, msg := range chatReq.Messages {
		var visioncontent []openai.VisionContent
		if err := json.Unmarshal(msg.Content, &visioncontent); err != nil {
			prompt += "<" + msg.Role + ">: " + string(msg.Content) + "\n"
			prompts = append(prompts, genai.Text("<"+msg.Role+">: "+string(msg.Content)))
		} else {
			if len(visioncontent) > 0 {
				for _, content := range visioncontent {
					if content.Type == "text" {
						prompt += "<" + msg.Role + ">: " + content.Text + "\n"
						prompts = append(prompts, genai.Text("<"+msg.Role+">: "+content.Text))
					} else if content.Type == "image_url" {
						if strings.HasPrefix(content.ImageURL.URL, "http") {
							fmt.Println("链接:", content.ImageURL.URL)
						} else if strings.HasPrefix(content.ImageURL.URL, "data:image") {
							fmt.Println("base64:", content.ImageURL.URL[:20])
							if chatReq.Model != "gemini-pro-vision" {
								chatReq.Model = "gemini-pro-vision"
							}

							var mime string
							// openai 会以 data:image 开头，则去掉 data:image/png;base64, 和 data:image/jpeg;base64,
							if strings.HasPrefix(content.ImageURL.URL, "data:image/png") {
								mime = "image/png"
							} else if strings.HasPrefix(content.ImageURL.URL, "data:image/jpeg") {
								mime = "image/jpeg"
							} else {
								c.JSON(http.StatusInternalServerError, gin.H{"error": "Unsupported image format"})
								return
							}
							imageString := strings.Split(content.ImageURL.URL, ",")[1]
							imageBytes, err := base64.StdEncoding.DecodeString(imageString)
							if err != nil {
								c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
								return
							}
							prompts = append(prompts, genai.Blob{MIMEType: mime, Data: imageBytes})
						}

						// todo image tokens
					}

				}

			}
		}
		if len(chatReq.Tools) > 0 {
			tooljson, _ := json.Marshal(chatReq.Tools)
			prompt += "<tools>: " + string(tooljson) + "\n"

			// for _, tool := range chatReq.Tools {

			// }

		}
	}

	usagelog.PromptCount = tokenizer.NumTokensFromStr(prompt, chatReq.Model)

	onekey, err := store.SelectKeyCache("google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(onekey.Key))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	model := client.GenerativeModel(chatReq.Model)

	iter := model.GenerateContentStream(ctx, prompts...)
	datachan := make(chan string)
	// closechan := make(chan error)
	var result string
	go func() {
		for {
			resp, err := iter.Next()
			if err == iterator.Done {

				var chatResp openai.ChatCompletionStreamResponse
				chatResp.Model = chatReq.Model
				choice := openai.Choice{}
				choice.FinishReason = "stop"
				chatResp.Choices = append(chatResp.Choices, choice)
				datachan <- "data: " + string(chatResp.ByteJson())
				close(datachan)
				break
			}
			if err != nil {
				log.Println(err)
				var errResp openai.ErrResponse
				errResp.Error.Code = "500"
				errResp.Error.Message = err.Error()
				datachan <- string(errResp.ByteJson())
				close(datachan)
				break
			}
			var content string
			if resp.Candidates != nil && len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
				if s, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
					content = string(s)
					result += content
				}
			} else {
				continue
			}

			var chatResp openai.ChatCompletionStreamResponse
			chatResp.Model = chatReq.Model
			choice := openai.Choice{}
			choice.Delta.Role = resp.Candidates[0].Content.Role
			choice.Delta.Content = content
			chatResp.Choices = append(chatResp.Choices, choice)

			chunk := "data: " + string(chatResp.ByteJson()) + "\n\n"
			datachan <- chunk
		}
	}()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		if data, ok := <-datachan; ok {
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

		}()
		return false
	})
}
