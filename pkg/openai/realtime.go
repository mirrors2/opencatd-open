/*
https://platform.openai.com/docs/guides/realtime
https://learn.microsoft.com/zh-cn/azure/ai-services/openai/how-to/audio-real-time

wss://my-eastus2-openai-resource.openai.azure.com/openai/realtime?api-version=2024-10-01-preview&deployment=gpt-4o-realtime-preview-1001
*/
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"opencatd-open/pkg/tokenizer"
	"opencatd-open/store"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

// "wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview-2024-10-01"
const realtimeURL = "wss://api.openai.com/v1/realtime"
const azureRealtimeURL = "wss://%s.openai.azure.com/openai/realtime?api-version=2024-10-01-preview&deployment=gpt-4o-realtime-preview"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type     string   `json:"type"`
	Response Response `json:"response"`
}

type Response struct {
	Modalities   []string `json:"modalities"`
	Instructions string   `json:"instructions"`
}

type RealTimeResponse struct {
	Type     string `json:"type"`
	EventID  string `json:"event_id"`
	Response struct {
		Object        string `json:"object"`
		ID            string `json:"id"`
		Status        string `json:"status"`
		StatusDetails any    `json:"status_details"`
		Output        []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Type    string `json:"type"`
			Status  string `json:"status"`
			Role    string `json:"role"`
			Content []struct {
				Type       string `json:"type"`
				Transcript string `json:"transcript"`
			} `json:"content"`
		} `json:"output"`
		Usage Usage `json:"usage"`
	} `json:"response"`
}

type Usage struct {
	TotalTokens       int `json:"total_tokens"`
	InputTokens       int `json:"input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	InputTokenDetails struct {
		CachedTokens int `json:"cached_tokens"`
		TextTokens   int `json:"text_tokens"`
		AudioTokens  int `json:"audio_tokens"`
	} `json:"input_token_details"`
	OutputTokenDetails struct {
		TextTokens  int `json:"text_tokens"`
		AudioTokens int `json:"audio_tokens"`
	} `json:"output_token_details"`
}

func RealTimeProxy(c *gin.Context) {
	log.Println(c.Request.URL.String())
	var model string = c.Query("model")
	value := url.Values{}
	value.Add("model", model)
	realtimeURL := realtimeURL + "?" + value.Encode()

	// 升级 HTTP 连接为 WebSocket
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer clientConn.Close()

	apikey, err := store.SelectKeyCacheByModel(model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 连接到 OpenAI WebSocket
	headers := http.Header{"OpenAI-Beta": []string{"realtime=v1"}}

	if apikey.ApiType == "azure" {
		headers.Set("api-key", apikey.Key)
		if apikey.EndPoint != "" {
			realtimeURL = fmt.Sprintf("%s/openai/realtime?api-version=2024-10-01-preview&deployment=gpt-4o-realtime-preview", apikey.EndPoint)
		} else {
			realtimeURL = fmt.Sprintf(azureRealtimeURL, apikey.ResourceNmae)
		}
	} else {
		headers.Set("Authorization", "Bearer "+apikey.Key)
	}

	conn := websocket.DefaultDialer
	if os.Getenv("LOCAL_PROXY") != "" {
		proxyUrl, _ := url.Parse(os.Getenv("LOCAL_PROXY"))
		conn.Proxy = http.ProxyURL(proxyUrl)
	}

	openAIConn, _, err := conn.Dial(realtimeURL, headers)
	if err != nil {
		log.Println("OpenAI dial error:", err)
		return
	}
	defer openAIConn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return forwardMessages(ctx, c, clientConn, openAIConn)
	})

	g.Go(func() error {
		return forwardMessages(ctx, c, openAIConn, clientConn)
	})

	if err := g.Wait(); err != nil {
		log.Println("Error in message forwarding:", err)
		return
	}

}

func forwardMessages(ctx context.Context, c *gin.Context, src, dst *websocket.Conn) error {
	usagelog := store.Tokens{Model: "gpt-4o-realtime-preview"}

	token, _ := c.Get("localuser")

	lu, err := store.GetUserByToken(token.(string))
	if err != nil {
		return err
	}
	usagelog.UserID = int(lu.ID)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			messageType, message, err := src.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return nil // 正常关闭，不报错
				}
				return err
			}
			if messageType == websocket.TextMessage {
				var usage Usage
				err := json.Unmarshal(message, &usage)
				if err == nil {
					usagelog.PromptCount += usage.InputTokens
					usagelog.CompletionCount += usage.OutputTokens
				}

			}
			err = dst.WriteMessage(messageType, message)
			if err != nil {
				return err
			}
		}
	}
	defer func() {
		usagelog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(usagelog.Model, usagelog.PromptCount, usagelog.CompletionCount))
		if err := store.Record(&usagelog); err != nil {
			log.Println(err)
		}
		if err := store.SumDaily(usagelog.UserID); err != nil {
			log.Println(err)
		}
	}()
	return nil
}
