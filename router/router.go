package router

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"opencatd-open/pkg/azureopenai"
	"opencatd-open/pkg/claude"
	oai "opencatd-open/pkg/openai"
	"opencatd-open/pkg/tokenizer"
	"opencatd-open/store"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sakurasan/to"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/vansante/go-ffprobe.v2"
	"gorm.io/gorm"
)

var (
	rootToken     string
	baseUrl       = "https://api.openai.com"
	GPT3Dot5Turbo = "gpt-3.5-turbo"
	GPT4          = "gpt-4"
	client        = getHttpClient()
)

type User struct {
	IsDelete  bool   `json:"IsDelete,omitempty"`
	ID        int    `json:"id,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Name      string `json:"name,omitempty"`
	Token     string `json:"token,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type Key struct {
	ID        int    `json:"id,omitempty"`
	Key       string `json:"key,omitempty"`
	Name      string `json:"name,omitempty"`
	ApiType   string `json:"api_type,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type ChatCompletionRequest struct {
	Model            string                  `json:"model"`
	Messages         []ChatCompletionMessage `json:"messages"`
	MaxTokens        int                     `json:"max_tokens,omitempty"`
	Temperature      float32                 `json:"temperature,omitempty"`
	TopP             float32                 `json:"top_p,omitempty"`
	N                int                     `json:"n,omitempty"`
	Stream           bool                    `json:"stream,omitempty"`
	Stop             []string                `json:"stop,omitempty"`
	PresencePenalty  float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32                 `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int          `json:"logit_bias,omitempty"`
	User             string                  `json:"user,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func init() {
	if openai_endpoint := os.Getenv("openai_endpoint"); openai_endpoint != "" {
		log.Println(fmt.Sprintf("replace %s to %s", baseUrl, openai_endpoint))
		baseUrl = openai_endpoint
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rootToken == "" {
			u, err := store.GetUserByID(uint(1))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			rootToken = u.Token
		}
		token := c.GetHeader("Authorization")
		if token == "" || token[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		if store.IsExistAuthCache(token[7:]) {
			if strings.HasPrefix(c.Request.URL.Path, "/1/me") {
				c.Next()
				return
			}
		}
		if token[7:] != rootToken {
			u, err := store.GetUserByID(uint(1))
			if err != nil {
				log.Println(err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			if token[:7] != u.Token {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			rootToken = u.Token
			store.LoadAuthCache()
		}
		// 可以在这里对 token 进行验证并检查权限

		c.Next()
	}
}

func Handleinit(c *gin.Context) {
	user, err := store.GetUserByID(1)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			u := store.User{Name: "root", Token: uuid.NewString()}
			u.ID = 1
			if err := store.CreateUser(&u); err != nil {
				c.JSON(http.StatusForbidden, gin.H{
					"error": err.Error(),
				})
				return
			} else {
				rootToken = u.Token
				resJSON := User{
					false,
					int(u.ID),
					u.UpdatedAt.Format(time.RFC3339),
					u.Name,
					u.Token,
					u.CreatedAt.Format(time.RFC3339),
				}
				c.JSON(http.StatusOK, resJSON)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	if user.ID == uint(1) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "super user already exists, use cli to reset password",
		})
	}
}

func HandleMe(c *gin.Context) {
	token := c.GetHeader("Authorization")
	u, err := store.GetUserByToken(token[7:])
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	resJSON := User{
		false,
		int(u.ID),
		u.UpdatedAt.Format(time.RFC3339),
		u.Name,
		u.Token,
		u.CreatedAt.Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, resJSON)
}

func HandleMeUsage(c *gin.Context) {
	token := c.GetHeader("Authorization")
	fromStr := c.Query("from")
	toStr := c.Query("to")
	getMonthStartAndEnd := func() (start, end string) {
		loc, _ := time.LoadLocation("Local")
		now := time.Now().In(loc)

		year, month, _ := now.Date()

		startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		start = startOfMonth.Format("2006-01-02")
		end = endOfMonth.Format("2006-01-02")
		return
	}
	if fromStr == "" || toStr == "" {
		fromStr, toStr = getMonthStartAndEnd()
	}
	user, err := store.GetUserByToken(token)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}
	usage, err := store.QueryUserUsage(to.String(user.ID), fromStr, toStr)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(200, usage)
}

func HandleKeys(c *gin.Context) {
	keys, err := store.GetAllKeys()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, keys)
}

func HandleUsers(c *gin.Context) {
	users, err := store.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, users)
}

func HandleAddKey(c *gin.Context) {
	var body Key
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": err.Error(),
		}})
		return
	}
	body.Name = strings.ToLower(strings.TrimSpace(body.Name))
	body.Key = strings.TrimSpace(body.Key)
	if strings.HasPrefix(body.Name, "azure.") {
		keynames := strings.Split(body.Name, ".")
		if len(keynames) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "Invalid Key Name",
			}})
			return
		}
		k := &store.Key{
			ApiType:      "azure",
			Name:         body.Name,
			Key:          body.Key,
			ResourceNmae: keynames[1],
			EndPoint:     body.Endpoint,
		}
		if err := store.CreateKey(k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": err.Error(),
			}})
			return
		}
	} else if strings.HasPrefix(body.Name, "claude.") {
		keynames := strings.Split(body.Name, ".")
		if len(keynames) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "Invalid Key Name",
			}})
			return
		}
		if body.Endpoint == "" {
			body.Endpoint = "https://api.anthropic.com"
		}
		k := &store.Key{
			// ApiType:      "anthropic",
			ApiType:      "claude",
			Name:         body.Name,
			Key:          body.Key,
			ResourceNmae: keynames[1],
			EndPoint:     body.Endpoint,
		}
		if err := store.CreateKey(k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": err.Error(),
			}})
			return
		}
	} else {
		if body.ApiType == "" {
			if err := store.AddKey("openai", body.Key, body.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
					"message": err.Error(),
				}})
				return
			}
		} else {
			k := &store.Key{
				ApiType:      body.ApiType,
				Name:         body.Name,
				Key:          body.Key,
				ResourceNmae: azureopenai.GetResourceName(body.Endpoint),
				EndPoint:     body.Endpoint,
			}
			if err := store.CreateKey(k); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
					"message": err.Error(),
				}})
				return
			}
		}

	}

	k, err := store.GetKeyrByName(body.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"message": err.Error(),
		}})
		return
	}
	c.JSON(http.StatusOK, k)
}

func HandleDelKey(c *gin.Context) {
	id := to.Int(c.Param("id"))
	if id < 1 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid key id"})
		return
	}
	if err := store.DeleteKey(uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "invalid key id"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func HandleAddUser(c *gin.Context) {
	var body User
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	if len(body.Name) == 0 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid user name"})
		return
	}

	if err := store.AddUser(body.Name, uuid.NewString()); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	u, err := store.GetUserByName(body.Name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func HandleDelUser(c *gin.Context) {
	id := to.Int(c.Param("id"))
	if id <= 1 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid user id"})
		return
	}
	if err := store.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func HandleResetUserToken(c *gin.Context) {
	id := to.Int(c.Param("id"))
	newtoken := c.Query("token")
	if newtoken == "" {
		newtoken = uuid.NewString()
	}

	if err := store.UpdateUser(uint(id), newtoken); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	u, err := store.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if u.ID == uint(1) {
		rootToken = u.Token
	}
	c.JSON(http.StatusOK, u)
}

func GenerateToken() string {
	token := uuid.New()
	return token.String()
}

func getHttpClient() *http.Client {
	tr := &http.Transport{
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
	}
	return &http.Client{Transport: tr}
}

func HandleProy(c *gin.Context) {
	var (
		localuser  bool
		isStream   bool
		chatreq    = openai.ChatCompletionRequest{}
		chatres    = openai.ChatCompletionResponse{}
		chatlog    store.Tokens
		onekey     store.Key
		pre_prompt string
		req        *http.Request
		err        error
		// wg         sync.WaitGroup
	)
	auth := c.Request.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		localuser = store.IsExistAuthCache(auth[7:])
		c.Set("localuser", auth[7:])
	}
	if c.Request.URL.Path == "/v1/complete" {
		claude.ClaudeProxy(c)
		return
	}
	if c.Request.URL.Path == "/v1/audio/transcriptions" {
		WhisperProxy(c)
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

	if c.Request.URL.Path == "/v1/chat/completions" && localuser {
		if store.KeysCache.ItemCount() == 0 {
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{
				"message": "No Api-Key Available",
			}})
			return
		}

		if err := c.BindJSON(&chatreq); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		chatlog.Model = chatreq.Model
		for _, m := range chatreq.Messages {
			pre_prompt += m.Content + "\n"
		}
		chatlog.PromptHash = cryptor.Md5String(pre_prompt)
		chatlog.PromptCount = tokenizer.NumTokensFromMessages(chatreq.Messages, chatreq.Model)
		isStream = chatreq.Stream
		chatlog.UserID, _ = store.GetUserID(auth[7:])

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(chatreq)

		if strings.HasPrefix(chatreq.Model, "claude-") {
			onekey, err = store.SelectKeyCache("claude")
			if err != nil {
				c.AbortWithError(http.StatusForbidden, err)
			}
		} else {
			onekey = store.FromKeyCacheRandomItemKey()
		}

		// 创建 API 请求
		switch onekey.ApiType {
		case "claude":
			payload, _ := claude.TransReq(&chatreq)
			buildurl := "https://api.anthropic.com/v1/complete"
			req, err = http.NewRequest("POST", buildurl, payload)
			req.Header.Add("accept", "application/json")
			req.Header.Add("anthropic-version", "2023-06-01")
			req.Header.Add("x-api-key", onekey.Key)
			req.Header.Add("content-type", "application/json")
		case "azure":
			fallthrough
		case "azure_openai":
			var buildurl string
			var apiVersion = "2023-05-15"
			if onekey.EndPoint != "" {
				buildurl = fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", onekey.EndPoint, modelmap(chatreq.Model), apiVersion)
			} else {
				buildurl = fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/chat/completions?api-version=%s", onekey.ResourceNmae, modelmap(chatreq.Model), apiVersion)
			}
			req, err = http.NewRequest(c.Request.Method, buildurl, &body)
			req.Header = c.Request.Header
			req.Header.Set("api-key", onekey.Key)
		case "openai":
			fallthrough
		default:
			if onekey.EndPoint != "" {
				req, err = http.NewRequest(c.Request.Method, onekey.EndPoint+c.Request.RequestURI, &body)
			} else {
				req, err = http.NewRequest(c.Request.Method, baseUrl+c.Request.RequestURI, &body)
			}

			req.Header = c.Request.Header
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", onekey.Key))
		}
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusOK, gin.H{"error": err.Error()})
			return
		}

	} else {
		req, err = http.NewRequest(c.Request.Method, baseUrl+c.Request.RequestURI, c.Request.Body)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		req.Header = c.Request.Header
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// 复制 API 响应头部
	for name, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	head := map[string]string{
		"Cache-Control":                    "no-store",
		"access-control-allow-origin":      "*",
		"access-control-allow-credentials": "true",
	}
	for k, v := range head {
		if _, ok := resp.Header[k]; !ok {
			c.Writer.Header().Set(k, v)
		}
	}
	resp.Header.Del("content-security-policy")
	resp.Header.Del("content-security-policy-report-only")
	resp.Header.Del("clear-site-data")

	c.Writer.WriteHeader(resp.StatusCode)
	writer := bufio.NewWriter(c.Writer)
	defer writer.Flush()

	reader := bufio.NewReader(resp.Body)

	if resp.StatusCode == 200 && localuser {
		switch onekey.ApiType {
		case "claude":
			claude.TransRsp(c, isStream, chatlog, reader)
			return
		case "openai", "azure", "azure_openai":
			fallthrough
		default:
			if isStream {
				contentCh := fetchResponseContent(c, reader)
				var buffer bytes.Buffer
				for content := range contentCh {
					buffer.WriteString(content)
				}
				chatlog.CompletionCount = tokenizer.NumTokensFromStr(buffer.String(), chatreq.Model)
				chatlog.TotalTokens = chatlog.PromptCount + chatlog.CompletionCount
				chatlog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
				if err := store.Record(&chatlog); err != nil {
					log.Println(err)
				}
				if err := store.SumDaily(chatlog.UserID); err != nil {
					log.Println(err)
				}
				return
			}
			res, err := io.ReadAll(reader)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
					"message": err.Error(),
				}})
				return
			}
			reader = bufio.NewReader(bytes.NewBuffer(res))
			json.NewDecoder(bytes.NewBuffer(res)).Decode(&chatres)
			chatlog.PromptCount = chatres.Usage.PromptTokens
			chatlog.CompletionCount = chatres.Usage.CompletionTokens
			chatlog.TotalTokens = chatres.Usage.TotalTokens
			chatlog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
			if err := store.Record(&chatlog); err != nil {
				log.Println(err)
			}
			if err := store.SumDaily(chatlog.UserID); err != nil {
				log.Println(err)
			}
		}

	}
	// 返回 API 响应主体
	if _, err := io.Copy(writer, reader); err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"message": err.Error(),
		}})
		return
	}
}

func HandleReverseProxy(c *gin.Context) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = "api.openai.com"
			// req.Header.Set("Authorization", "Bearer YOUR_API_KEY_HERE")
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

	var localuser bool
	auth := c.Request.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		log.Println(store.IsExistAuthCache(auth[7:]))
		localuser = store.IsExistAuthCache(auth[7:])
	}

	req, err := http.NewRequest(c.Request.Method, c.Request.URL.Path, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	req.Header = c.Request.Header
	if localuser {
		if store.KeysCache.ItemCount() == 0 {
			c.JSON(http.StatusOK, gin.H{"error": "No Api-Key Available"})
			return
		}
		onekey := store.FromKeyCacheRandomItemKey()
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", onekey.Key))
	}

	proxy.ServeHTTP(c.Writer, req)

}

func HandleUsage(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	getMonthStartAndEnd := func() (start, end string) {
		loc, _ := time.LoadLocation("Local")
		now := time.Now().In(loc)

		year, month, _ := now.Date()

		startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		start = startOfMonth.Format("2006-01-02")
		end = endOfMonth.Format("2006-01-02")
		return
	}
	if fromStr == "" || toStr == "" {
		fromStr, toStr = getMonthStartAndEnd()
	}

	usage, err := store.QueryUsage(fromStr, toStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, usage)
}

func fetchResponseContent(ctx *gin.Context, responseBody *bufio.Reader) <-chan string {
	contentCh := make(chan string)
	go func() {
		defer close(contentCh)
		for {
			line, err := responseBody.ReadString('\n')
			if err == nil {
				lines := strings.Split(line, "")
				for _, word := range lines {
					ctx.Writer.WriteString(word)
					ctx.Writer.Flush()
				}
				if line == "\n" {
					continue
				}
				if strings.HasPrefix(line, "data:") {
					line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
					if strings.HasSuffix(line, "[DONE]") {
						break
					}
					line = strings.TrimSpace(line)
				}

				dec := json.NewDecoder(strings.NewReader(line))
				var data map[string]interface{}
				if err := dec.Decode(&data); err == io.EOF {
					log.Println("EOF:", err)
					break
				} else if err != nil {
					fmt.Println("Error decoding response:", err)
					return
				}
				if choices, ok := data["choices"].([]interface{}); ok {
					for _, choice := range choices {
						choiceMap := choice.(map[string]interface{})
						if content, ok := choiceMap["delta"].(map[string]interface{})["content"]; ok {
							contentCh <- content.(string)
						}
					}
				}
			} else {
				break
			}
		}
	}()
	return contentCh
}

func modelmap(in string) string {
	// gpt-3.5-turbo -> gpt-35-turbo
	if strings.Contains(in, ".") {
		return strings.ReplaceAll(in, ".", "")
	}
	return in
}

func WhisperProxy(c *gin.Context) {
	var chatlog store.Tokens

	byteBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(byteBody))

	model, _ := c.GetPostForm("model")

	key, err := store.SelectKeyCache("openai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
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

	if err := ParseWhisperRequestTokens(c, &chatlog, byteBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}
	if key.EndPoint == "" {
		key.EndPoint = "https://api.openai.com"
	}
	targetUrl, _ := url.ParseRequestURI(key.EndPoint + c.Request.URL.String())
	log.Println(targetUrl)
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Director = func(req *http.Request) {
		req.Host = targetUrl.Host
		req.URL.Scheme = targetUrl.Scheme
		req.URL.Host = targetUrl.Host

		req.Header.Set("Authorization", "Bearer "+key.Key)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			return nil
		}
		chatlog.TotalTokens = chatlog.PromptCount + chatlog.CompletionCount
		chatlog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
		if err := store.Record(&chatlog); err != nil {
			log.Println(err)
		}
		if err := store.SumDaily(chatlog.UserID); err != nil {
			log.Println(err)
		}
		return nil
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func probe(fileReader io.Reader) (time.Duration, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	data, err := ffprobe.ProbeReader(ctx, fileReader)
	if err != nil {
		return 0, err
	}

	duration := data.Format.DurationSeconds
	pduration, err := time.ParseDuration(fmt.Sprintf("%fs", duration))
	if err != nil {
		return 0, fmt.Errorf("Error parsing duration: %s", err)
	}
	return pduration, nil
}

func getAudioDuration(file *multipart.FileHeader) (time.Duration, error) {
	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
		err      error
	)

	f, err := file.Open()
	defer f.Close()

	// Get the file extension to determine the audio file type
	fileType := filepath.Ext(file.Filename)

	switch fileType {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	case ".m4a":
		duration, err := probe(f)
		if err != nil {
			return 0, err
		}
		return duration, nil
	default:
		return 0, errors.New("unsupported audio file format")
	}

	if err != nil {
		return 0, err
	}
	defer streamer.Close()

	// Calculate the audio file's duration.
	numSamples := streamer.Len()
	sampleRate := format.SampleRate
	duration := time.Duration(numSamples) * time.Second / time.Duration(sampleRate)

	return duration, nil
}

func ParseWhisperRequestTokens(c *gin.Context, usage *store.Tokens, byteBody []byte) error {
	file, _ := c.FormFile("file")
	model, _ := c.GetPostForm("model")
	usage.Model = model

	if file != nil {
		duration, err := getAudioDuration(file)
		if err != nil {
			return fmt.Errorf("Error getting audio duration:%s", err)
		}

		if duration > 5*time.Minute {
			return fmt.Errorf("Audio duration exceeds 5 minutes")
		}
		// 计算时长，四舍五入到最接近的秒数
		usage.PromptCount = int(duration.Round(time.Second).Seconds())
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(byteBody))

	return nil
}
