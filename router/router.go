package router

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"opencatd-open/pkg/claude"
	oai "opencatd-open/pkg/openai"
	"opencatd-open/store"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	baseUrl       = "https://api.openai.com"
	GPT3Dot5Turbo = "gpt-3.5-turbo"
	GPT4          = "gpt-4"
)

// type ChatCompletionMessage struct {
// 	Role    string `json:"role"`
// 	Content string `json:"content"`
// 	Name    string `json:"name,omitempty"`
// }

// type ChatCompletionRequest struct {
// 	Model            string                  `json:"model"`
// 	Messages         []ChatCompletionMessage `json:"messages"`
// 	MaxTokens        int                     `json:"max_tokens,omitempty"`
// 	Temperature      float32                 `json:"temperature,omitempty"`
// 	TopP             float32                 `json:"top_p,omitempty"`
// 	N                int                     `json:"n,omitempty"`
// 	Stream           bool                    `json:"stream,omitempty"`
// 	Stop             []string                `json:"stop,omitempty"`
// 	PresencePenalty  float32                 `json:"presence_penalty,omitempty"`
// 	FrequencyPenalty float32                 `json:"frequency_penalty,omitempty"`
// 	LogitBias        map[string]int          `json:"logit_bias,omitempty"`
// 	User             string                  `json:"user,omitempty"`
// }

// type ChatCompletionChoice struct {
// 	Index        int                   `json:"index"`
// 	Message      ChatCompletionMessage `json:"message"`
// 	FinishReason string                `json:"finish_reason"`
// }

// type ChatCompletionResponse struct {
// 	ID      string                 `json:"id"`
// 	Object  string                 `json:"object"`
// 	Created int64                  `json:"created"`
// 	Model   string                 `json:"model"`
// 	Choices []ChatCompletionChoice `json:"choices"`
// 	Usage   struct {
// 		PromptTokens     int `json:"prompt_tokens"`
// 		CompletionTokens int `json:"completion_tokens"`
// 		TotalTokens      int `json:"total_tokens"`
// 	} `json:"usage"`
// }

func init() {
	if openai_endpoint := os.Getenv("openai_endpoint"); openai_endpoint != "" {
		log.Println(fmt.Sprintf("replace %s to %s", baseUrl, openai_endpoint))
		baseUrl = openai_endpoint
	}
}

func HandleProxy(c *gin.Context) {
	var (
		localuser bool
	)
	auth := c.Request.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		localuser = store.IsExistAuthCache(auth[7:])
		c.Set("localuser", auth[7:])
	}
	if c.Request.URL.Path == "/v1/complete" {
		if localuser {
			claude.ClaudeProxy(c)
			return
		} else {
			HandleReverseProxy(c, "api.anthropic.com")
			return
		}

	}
	if c.Request.URL.Path == "/v1/audio/transcriptions" {
		oai.WhisperProxy(c)
		return
	}
	if c.Request.URL.Path == "/v1/audio/speech" {
		oai.SpeechHandler(c)
		return
	}

	if c.Request.URL.Path == "/v1/images/generations" {
		oai.DalleHandler(c)
		return
	}

	if c.Request.URL.Path == "/v1/chat/completions" {
		if localuser {
			if store.KeysCache.ItemCount() == 0 {
				c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{
					"message": "No Api-Key Available",
				}})
				return
			}

			ChatHandler(c)
			return
		}
	} else {
		HandleReverseProxy(c, "api.openai.com")
		return
	}

}

func HandleReverseProxy(c *gin.Context, targetHost string) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = targetHost
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest(c.Request.Method, c.Request.URL.Path, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	req.Header = c.Request.Header

	proxy.ServeHTTP(c.Writer, req)
	return
}
