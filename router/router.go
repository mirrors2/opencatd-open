package router

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"opencatd-open/pkg/azureopenai"
	"opencatd-open/store"
	"os"
	"strings"
	"time"

	"github.com/Sakurasan/to"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
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
			ApiType:      "azure_openai",
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

	if err := store.UpdateUser(uint(id), uuid.NewString()); err != nil {
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

func sendRequestWithRetry(c *gin.Context, client *http.Client, req *http.Request, maxRetries int, seed int64, initialKey store.Key) (*http.Response, error) {
	var resp *http.Response
	var err error

	onekey := initialKey // 初始化onekey为传入的初始值

	// 读取请求体
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	req.Body.Close() // 记得关闭 req.Body

	type Error struct {
		Message string `json:"message"`
	}

	type ResponseBody struct {
		Error Error `json:"error"`
	}

	for i := 0; i < maxRetries; i++ {
		reqCopy := req.Clone(c) // 克隆请求

		if i > 0 {
			onekey = store.FromKeyCacheRandomItemKey(i, seed)                         // 重试时获取新的键值
			reqCopy.Header.Set("Authorization", fmt.Sprintf("Bearer %s", onekey.Key)) // 只有在重试时才更改请求头
		}

		// 使用读取的请求体
		reqCopy.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		resp, err = client.Do(reqCopy)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		// 如果状态码是429, 则需要进行等待或者直接重试
		if resp.StatusCode == 429 {
			// 读取响应体
			respBodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				return nil, err
			}

			// 将响应体数据写回以便外部访问
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyBytes))

			// 解析响应体
			var responseBody ResponseBody
			err = json.Unmarshal(respBodyBytes, &responseBody)
			if err != nil {
				log.Println(err)
				return nil, err
			}

			// 检查响应体中的message是否等于 "rate limited"
			if responseBody.Error.Message == "rate limited" {
				// 如果是，则等待5秒再重试
				log.Printf("Received 429 status code and 'rate limited' message for retry %d using key %s with name %s and api-type %s. Sleeping for 5 seconds.", i, onekey.Key, onekey.Name, onekey.ApiType)
				time.Sleep(5 * time.Second)
			} else {
				// 如果响应体中的message不是 "rate limited", 则直接重试
				log.Printf("Received 429 status code for retry %d using key %s with name %s and api-type %s. Retrying immediately.", i, onekey.Key, onekey.Name, onekey.ApiType)
			}

			continue
		}

		// 如果状态码不是429，我们得到了一个有效的响应，所以跳出循环并返回响应和nil错误
		break
	}

	return resp, nil // 最后返回响应和nil，而不是错误信息
}

func HandleProxy(c *gin.Context) {
	var (
		localuser  bool
		isStream   bool
		chatreq    = openai.ChatCompletionRequest{}
		chatres    = openai.ChatCompletionResponse{}
		chatlog    store.Tokens
		pre_prompt string
		req        *http.Request
		err        error
		// wg         sync.WaitGroup
	)
	seed := time.Now().UnixNano() / 1e6
	auth := c.Request.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		localuser = store.IsExistAuthCache(auth[7:])
	}

	onekey := store.FromKeyCacheRandomItemKey(0, seed)

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
		chatlog.PromptCount = NumTokensFromMessages(chatreq.Messages, chatreq.Model)
		isStream = chatreq.Stream
		chatlog.UserID, _ = store.GetUserID(auth[7:])

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(chatreq)
		// 创建 API 请求
		switch onekey.ApiType {
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
		case "openai", "openai-plus":
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
			c.JSON(http.StatusOK, gin.H{"error": err.Error()})
			return
		}
		req.Header = c.Request.Header
	}

	maxRetries := 5 // 设置最大重试次数为5
	resp, err := sendRequestWithRetry(c, client, req, maxRetries, seed, onekey)

	if err != nil {
		log.Println(err)
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
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
		if isStream {
			contentCh := fetchResponseContent(c, reader)
			var buffer bytes.Buffer
			for content := range contentCh {
				buffer.WriteString(content)
			}
			chatlog.CompletionCount = NumTokensFromStr(buffer.String(), chatreq.Model)
			chatlog.TotalTokens = chatlog.PromptCount + chatlog.CompletionCount
			chatlog.Cost = fmt.Sprintf("%.6f", Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
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
		chatlog.Cost = fmt.Sprintf("%.6f", Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
		if err := store.Record(&chatlog); err != nil {
			log.Println(err)
		}
		if err := store.SumDaily(chatlog.UserID); err != nil {
			log.Println(err)
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

func Cost(model string, promptCount, completionCount int) float64 {
	var cost, prompt, completion float64
	prompt = float64(promptCount)
	completion = float64(completionCount)

	switch model {
	case "gpt-3.5-turbo", "gpt-3.5-turbo-0301":
		cost = 0.002 * float64((prompt+completion)/1000)
	case "gpt-4", "gpt-4-0314":
		cost = 0.03*float64(prompt/1000) + 0.06*float64(completion/1000)
	case "gpt-4-32k", "gpt-4-32k-0314":
		cost = 0.06*float64(prompt/1000) + 0.12*float64(completion/1000)
	}
	return cost
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

func NumTokensFromMessages(messages []openai.ChatCompletionMessage, model string) (num_tokens int) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		err = fmt.Errorf("EncodingForModel: %v", err)
		fmt.Println(err)
		return
	}

	var tokens_per_message int
	var tokens_per_name int
	if model == "gpt-3.5-turbo-0301" || model == "gpt-3.5-turbo" {
		tokens_per_message = 4
		tokens_per_name = -1
	} else if model == "gpt-4-0314" || model == "gpt-4" {
		tokens_per_message = 3
		tokens_per_name = 1
	} else {
		fmt.Println("Warning: model not found. Using cl100k_base encoding.")
		tokens_per_message = 3
		tokens_per_name = 1
	}

	for _, message := range messages {
		num_tokens += tokens_per_message
		num_tokens += len(tkm.Encode(message.Content, nil, nil))
		// num_tokens += len(tkm.Encode(message.Role, nil, nil))
		if message.Name != "" {
			num_tokens += tokens_per_name
		}
	}
	num_tokens += 3
	return num_tokens
}

func NumTokensFromStr(messages string, model string) (num_tokens int) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		err = fmt.Errorf("EncodingForModel: %v", err)
		fmt.Println(err)
		return
	}

	num_tokens += len(tkm.Encode(messages, nil, nil))
	return num_tokens
}

func modelmap(in string) string {
	switch in {
	case "gpt-3.5-turbo":
		return "gpt-35-turbo"
	case "gpt-4":
		return "gpt-4"
	}
	return in
}
