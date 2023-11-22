package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"opencatd-open/pkg/tokenizer"
	"opencatd-open/store"

	"github.com/gin-gonic/gin"
)

const (
	SpeechEndpoint = "https://api.openai.com/v1/audio/speech"
)

type SpeechRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

func SpeechHandler(c *gin.Context) {
	var chatreq SpeechRequest
	if err := c.ShouldBindJSON(&chatreq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var chatlog store.Tokens
	chatlog.Model = chatreq.Model
	chatlog.CompletionCount = len(chatreq.Input)

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
	chatlog.UserID = int(lu.ID)

	key, err := store.SelectKeyCache("openai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}

	targetURL, _ := url.Parse(SpeechEndpoint)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Header["Authorization"] = []string{"Bearer " + key.Key}
		req.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path
		req.URL.RawPath = targetURL.RawPath

		reqBytes, _ := json.Marshal(chatreq)
		req.Body = io.NopCloser(bytes.NewReader(reqBytes))
		req.ContentLength = int64(len(reqBytes))

	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode == http.StatusOK {
			chatlog.TotalTokens = chatlog.PromptCount + chatlog.CompletionCount
			chatlog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
			if err := store.Record(&chatlog); err != nil {
				log.Println(err)
			}
			if err := store.SumDaily(chatlog.UserID); err != nil {
				log.Println(err)
			}
		}
		return nil
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}
