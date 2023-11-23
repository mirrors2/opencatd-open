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
	"strconv"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gin-gonic/gin"
)

const (
	DalleEndpoint          = "https://api.openai.com/v1/images/generations"
	DalleEditEndpoint      = "https://api.openai.com/v1/images/edits"
	DalleVariationEndpoint = "https://api.openai.com/v1/images/variations"
)

type DallERequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `form:"n" json:"n,omitempty"`
	Size           string `form:"size" json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`         // standard,hd
	Style          string `json:"style,omitempty"`           // vivid,natural
	ResponseFormat string `json:"response_format,omitempty"` // url or b64_json
}

func DalleHandler(c *gin.Context) {

	var dalleRequest DallERequest
	if err := c.ShouldBind(&dalleRequest); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if dalleRequest.N == 0 {
		dalleRequest.N = 1
	}

	if dalleRequest.Size == "" {
		dalleRequest.Size = "512x512"
	}

	model := dalleRequest.Model

	var chatlog store.Tokens
	chatlog.CompletionCount = dalleRequest.N

	if model == "dall-e" {
		model = "dall-e-2"
	}
	model = model + "." + dalleRequest.Size

	if dalleRequest.Model == "dall-e-2" || dalleRequest.Model == "dall-e" {
		if !slice.Contain([]string{"256x256", "512x512", "1024x1024"}, dalleRequest.Size) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("Invalid size: %s for %s", dalleRequest.Size, dalleRequest.Model),
				},
			})
			return
		}
	} else if dalleRequest.Model == "dall-e-3" {
		if !slice.Contain([]string{"256x256", "512x512", "1024x1024", "1792x1024", "1024x1792"}, dalleRequest.Size) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("Invalid size: %s for %s", dalleRequest.Size, dalleRequest.Model),
				},
			})
			return
		}
		if dalleRequest.Quality == "hd" {
			model = model + ".hd"
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("Invalid model: %s", dalleRequest.Model),
			},
		})
		return
	}
	chatlog.Model = model

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

	targetURL, _ := url.Parse(DalleEndpoint)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+key.Key)
		req.Header.Set("Content-Type", "application/json")

		req.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path
		req.URL.RawPath = targetURL.RawPath
		req.URL.RawQuery = targetURL.RawQuery

		bytebody, _ := json.Marshal(dalleRequest)
		req.Body = io.NopCloser(bytes.NewBuffer(bytebody))
		req.ContentLength = int64(len(bytebody))
		req.Header.Set("Content-Length", strconv.Itoa(len(bytebody)))
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
