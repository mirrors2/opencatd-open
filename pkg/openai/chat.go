package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"opencatd-open/pkg/tokenizer"
	"opencatd-open/store"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AzureApiVersion = "2024-02-01"
	OpenAI_Endpoint = "https://api.openai.com/v1/chat/completions"
)

var (
	BaseURL            string // "https://api.openai.com"
	AIGateWay_Endpoint = "https://gateway.ai.cloudflare.com/v1/431ba10f11200d544922fbca177aaa7f/openai/openai/chat/completions"
)

func init() {
	if os.Getenv("OpenAI_Endpoint") != "" {
		BaseURL = os.Getenv("OpenAI_Endpoint")
	}
	if os.Getenv("AIGateWay_Endpoint") != "" {
		AIGateWay_Endpoint = os.Getenv("AIGateWay_Endpoint")
	}
}

// Vision Content
type VisionContent struct {
	Type     string          `json:"type,omitempty"`
	Text     string          `json:"text,omitempty"`
	ImageURL *VisionImageURL `json:"image_url,omitempty"`
}
type VisionImageURL struct {
	URL    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type ChatCompletionMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Name    string          `json:"name,omitempty"`
}

type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

type Tool struct {
	Type     string              `json:"type"`
	Function *FunctionDefinition `json:"function,omitempty"`
}

type ChatCompletionRequest struct {
	Model            string                  `json:"model"`
	Messages         []ChatCompletionMessage `json:"messages"`
	MaxTokens        int                     `json:"max_tokens,omitempty"`
	Temperature      float64                 `json:"temperature,omitempty"`
	TopP             float64                 `json:"top_p,omitempty"`
	N                int                     `json:"n,omitempty"`
	Stream           bool                    `json:"stream,omitempty"`
	Stop             []string                `json:"stop,omitempty"`
	PresencePenalty  float64                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64                 `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int          `json:"logit_bias,omitempty"`
	User             string                  `json:"user,omitempty"`
	// Functions        []FunctionDefinition       `json:"functions,omitempty"`
	// FunctionCall     any                        `json:"function_call,omitempty"`
	Tools []Tool `json:"tools,omitempty"`
	// ToolChoice any    `json:"tool_choice,omitempty"`
}

func (c ChatCompletionRequest) ToByteJson() []byte {
	bytejson, _ := json.Marshal(c)
	return bytejson
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
		Logprobs     string `json:"logprobs"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	SystemFingerprint string `json:"system_fingerprint"`
}

type Choice struct {
	Index int `json:"index"`
	Delta struct {
		Role      string     `json:"role"`
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
	Usage        struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ChatCompletionStreamResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

func (c *ChatCompletionStreamResponse) ByteJson() []byte {
	bytejson, _ := json.Marshal(c)
	return bytejson
}

func ChatProxy(c *gin.Context, chatReq *ChatCompletionRequest) {
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

	var prompt string
	for _, msg := range chatReq.Messages {
		// prompt += "<" + msg.Role + ">: " + msg.Content + "\n"
		var visioncontent []VisionContent
		if err := json.Unmarshal(msg.Content, &visioncontent); err != nil {
			prompt += "<" + msg.Role + ">: " + string(msg.Content) + "\n"
		} else {
			if len(visioncontent) > 0 {
				for _, content := range visioncontent {
					if content.Type == "text" {
						prompt += "<" + msg.Role + ">: " + content.Text + "\n"
					} else if content.Type == "image_url" {
						if strings.HasPrefix(content.ImageURL.URL, "http") {
							fmt.Println("链接:", content.ImageURL.URL)
						} else if strings.HasPrefix(content.ImageURL.URL, "data:image") {
							fmt.Println("base64:", content.ImageURL.URL[:20])
						}
						// todo image tokens
					}

				}

			}
		}
		if len(chatReq.Tools) > 0 {
			tooljson, _ := json.Marshal(chatReq.Tools)
			prompt += "<tools>: " + string(tooljson) + "\n"
		}
	}

	usagelog.PromptCount = tokenizer.NumTokensFromStr(prompt, chatReq.Model)

	onekey, err := store.SelectKeyCache("openai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req *http.Request

	switch onekey.ApiType {
	case "azure":
		var buildurl string
		if onekey.EndPoint != "" {
			buildurl = fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", onekey.EndPoint, modelmap(chatReq.Model), AzureApiVersion)
		} else {
			buildurl = fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/chat/completions?api-version=%s", onekey.ResourceNmae, modelmap(chatReq.Model), AzureApiVersion)
		}
		req, err = http.NewRequest(c.Request.Method, buildurl, bytes.NewReader(chatReq.ToByteJson()))
		req.Header = c.Request.Header
		req.Header.Set("api-key", onekey.Key)
	default:
		if onekey.EndPoint != "" { // 优先key的endpoint
			req, err = http.NewRequest(c.Request.Method, onekey.EndPoint+c.Request.RequestURI, bytes.NewReader(chatReq.ToByteJson()))
		} else {
			if BaseURL != "" { // 其次BaseURL
				req, err = http.NewRequest(c.Request.Method, BaseURL+c.Request.RequestURI, bytes.NewReader(chatReq.ToByteJson()))
			} else { // 最后是gateway的endpoint
				req, err = http.NewRequest(c.Request.Method, AIGateWay_Endpoint, bytes.NewReader(chatReq.ToByteJson()))
			}
		}
		req.Header = c.Request.Header
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", onekey.Key))
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	teeReader := io.TeeReader(resp.Body, c.Writer)

	var result string
	if chatReq.Stream {
		// 流式响应
		scanner := bufio.NewScanner(teeReader)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 && bytes.HasPrefix(line, []byte("data: ")) {
				if bytes.HasPrefix(line, []byte("data: [DONE]")) {
					break
				}
				var opiResp ChatCompletionStreamResponse
				line = bytes.Replace(line, []byte("data: "), []byte(""), -1)
				line = bytes.TrimSpace(line)
				if err := json.Unmarshal(line, &opiResp); err != nil {
					continue
				}

				if opiResp.Choices != nil && len(opiResp.Choices) > 0 {
					if opiResp.Choices[0].Delta.Role != "" {
						result += "<" + opiResp.Choices[0].Delta.Role + "> "
					}
					result += opiResp.Choices[0].Delta.Content // 计算Content Token

					if len(opiResp.Choices[0].Delta.ToolCalls) > 0 { // 计算ToolCalls token
						if opiResp.Choices[0].Delta.ToolCalls[0].Function.Name != "" {
							result += "name:" + opiResp.Choices[0].Delta.ToolCalls[0].Function.Name + " arguments:"
						}
						result += opiResp.Choices[0].Delta.ToolCalls[0].Function.Arguments
					}
				} else {
					continue
				}
			}

		}
	} else {
		// 处理非流式响应
		body, err := io.ReadAll(teeReader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var opiResp ChatCompletionResponse
		if err := json.Unmarshal(body, &opiResp); err != nil {
			log.Println("Error parsing JSON:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing JSON," + err.Error()})
			return
		}
		if opiResp.Choices != nil && len(opiResp.Choices) > 0 {
			if opiResp.Choices[0].Message.Role != "" {
				result += "<" + opiResp.Choices[0].Message.Role + "> "
			}
			result += opiResp.Choices[0].Message.Content

			if len(opiResp.Choices[0].Message.ToolCalls) > 0 {
				if opiResp.Choices[0].Message.ToolCalls[0].Function.Name != "" {
					result += "name:" + opiResp.Choices[0].Message.ToolCalls[0].Function.Name + " arguments:"
				}
				result += opiResp.Choices[0].Message.ToolCalls[0].Function.Arguments
			}

		}
	}
	usagelog.CompletionCount = tokenizer.NumTokensFromStr(result, chatReq.Model)
	usagelog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(usagelog.Model, usagelog.PromptCount, usagelog.CompletionCount))
	if err := store.Record(&usagelog); err != nil {
		log.Println(err)
	}
	if err := store.SumDaily(usagelog.UserID); err != nil {
		log.Println(err)
	}
}

func modelmap(in string) string {
	// gpt-3.5-turbo -> gpt-35-turbo
	if strings.Contains(in, ".") {
		return strings.ReplaceAll(in, ".", "")
	}
	return in
}

type ErrResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (e *ErrResponse) ByteJson() []byte {
	bytejson, _ := json.Marshal(e)
	return bytejson
}
