/*
https://platform.openai.com/docs/guides/realtime

wss://my-eastus2-openai-resource.openai.azure.com/openai/realtime?api-version=2024-10-01-preview&deployment=gpt-4o-realtime-preview-1001
*/
package openai

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

// "wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview-2024-10-01"
const realtimeURL = "wss://api.openai.com/v1/realtime"

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

	// 连接到 OpenAI WebSocket
	headers := http.Header{
		"Authorization": []string{"Bearer " + os.Getenv("OPENAI_API_KEY")},
		"OpenAI-Beta":   []string{"realtime=v1"},
	}

	conn := websocket.Dialer{
		// Proxy:            http.ProxyURL(&url.URL{Scheme: "http", Host: "127.0.0.1:7890"}),
		HandshakeTimeout: 45 * time.Second,
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
		return forwardMessages(ctx, clientConn, openAIConn)
	})

	g.Go(func() error {
		return forwardMessages(ctx, openAIConn, clientConn)
	})

	if err := g.Wait(); err != nil {
		log.Println("Error in message forwarding:", err)
		return
	}

}

func forwardMessages(ctx context.Context, src, dst *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, message, err := src.ReadMessage()
			if err != nil {
				return err
			}
			log.Println("Received message:", string(message))
			err = dst.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return err
			}
		}
	}
}
