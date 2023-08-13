/*
https://docs.anthropic.com/claude/reference/complete_post

curl --request POST \
     --url https://api.anthropic.com/v1/complete \
     --header "anthropic-version: 2023-06-01" \
     --header "content-type: application/json" \
     --header "x-api-key: $ANTHROPIC_API_KEY" \
     --data '
{
  "model": "claude-2",
  "prompt": "\n\nHuman: Hello, world!\n\nAssistant:",
  "max_tokens_to_sample": 256,
  "stream": true
}
'

{"completion":" Hello! Nice to meet you.","stop_reason":"stop_sequence","model":"claude-2.0","stop":"\n\nHuman:","log_id":"727bded01002627057967d02b3d557a01aa73266849b62f5aa0b97dec1247ed3"}

event: completion
data: {"completion":"","stop_reason":"stop_sequence","model":"claude-2.0","stop":"\n\nHuman:","log_id":"dfd42341ad08856ff01811885fb8640a1bf977551d8331f81fe9a6c8182c6c63"}

# Model Pricing

Claude Instant |100,000 tokens |Prompt $1.63/million tokens |Completion $5.51/million tokens

Claude 2 |100,000 tokens |Prompt $11.02/million tokens |Completion $32.68/million tokens
*Claude 1 is still accessible and offered at the same price as Claude 2.

*/

// package anthropic
package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"opencatd-open/store"
	"strings"

	"github.com/gin-gonic/gin"
)

type MessageModule struct {
	Assistant string // returned data (do not modify)
	Human     string // input content
}

type CompleteRequest struct {
	Model             string `json:"model,omitempty"`                //*
	Prompt            string `json:"prompt,omitempty"`               //*
	MaxTokensToSample int    `json:"max_Tokens_To_Sample,omitempty"` //*
	StopSequences     string `json:"stop_Sequences,omitempty"`
	Temperature       int    `json:"temperature,omitempty"`
	TopP              int    `json:"top_P,omitempty"`
	TopK              int    `json:"top_K,omitempty"`
	Stream            bool   `json:"stream,omitempty"`
	Metadata          struct {
		UserId string `json:"user_Id,omitempty"`
	} `json:"metadata,omitempty"`
}

type CompleteResponse struct {
	Completion string `json:"completion"`
	StopReason string `json:"stop_reason"`
	Model      string `json:"model"`
	Stop       string `json:"stop"`
	LogID      string `json:"log_id"`
}

func Create() {
	url := "https://api.anthropic.com/v1/complete"

	payload := strings.NewReader("{\"model\":\"claude-2\",\"prompt\":\"\\n\\nHuman: Hello, world!\\n\\nAssistant:\",\"max_tokens_to_sample\":256}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("anthropic-version", "2023-06-01")
	req.Header.Add("x-api-key", "$ANTHROPIC_API_KEY")
	req.Header.Add("content-type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(string(body))
}

func ClaudeProxy(c *gin.Context) {
	var chatlog store.Tokens
	var complete CompleteRequest

	byteBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(byteBody))

	if err := json.Unmarshal(byteBody, &complete); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	key, err := store.SelectKeyCache("anthropic")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}

	chatlog.Model = complete.Model

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

	// todo calc prompt token

	if key.EndPoint == "" {
		key.EndPoint = "https://api.anthropic.com"
	}
	targetUrl, _ := url.ParseRequestURI(key.EndPoint + c.Request.URL.String())

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Director = func(req *http.Request) {
		req.Host = targetUrl.Host
		req.URL.Scheme = targetUrl.Scheme
		req.URL.Host = targetUrl.Host

		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("content-type", "application/json")
		req.Header.Set("x-api-key", key.Key)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			return nil
		}
		var byteResp []byte
		byteResp, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(byteResp))
		if complete.Stream != true {
			var complete_resp CompleteResponse

			if err := json.Unmarshal(byteResp, &complete_resp); err != nil {
				log.Println(err)
				return nil
			}
		} else {
			var completion string
			for {
				line, err := bufio.NewReader(bytes.NewBuffer(byteResp)).ReadString('\n')
				if err != nil {
					if strings.HasPrefix(line, "data:") {
						line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
						if strings.HasSuffix(line, "[DONE]") {
							break
						}
						line = strings.TrimSpace(line)
						var complete_resp CompleteResponse
						if err := json.Unmarshal([]byte(line), &complete_resp); err != nil {
							log.Println(err)
							break
						}
						completion += line
					}
				}
			}
			log.Println("completion:", completion)
		}
		chatlog.TotalTokens = chatlog.PromptCount + chatlog.CompletionCount

		// todo calc cost

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
