/*
文档 https://www.microsoft.com/en-us/bing/apis/bing-web-search-api
价格 https://www.microsoft.com/en-us/bing/apis/pricing

curl -H "Ocp-Apim-Subscription-Key: <yourkeygoeshere>" https://api.bing.microsoft.com/v7.0/search?q=今天上海天气怎么样
curl -H "Ocp-Apim-Subscription-Key: 6fc7c97ebed54f75a5e383ee2272c917" https://api.bing.microsoft.com/v7.0/search?q=今天上海天气怎么样
*/

package search

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/tidwall/gjson"
)

const (
	bingEndpoint = "https://api.bing.microsoft.com/v7.0/search"
)

var subscriptionKey string

func init() {
	if os.Getenv("bing") != "" {
		subscriptionKey = os.Getenv("bing")
	} else {
		log.Println("bing key not found")
	}
}

func BingSearch(searchParams SearchParams) (any, error) {
	params := url.Values{}
	params.Set("q", searchParams.Query)
	params.Set("count", "5")
	if searchParams.Num > 0 {
		params.Set("count", fmt.Sprintf("%d", searchParams.Num))
	}

	reqURL, _ := url.Parse(bingEndpoint)
	reqURL.RawQuery = params.Encode()

	req, _ := http.NewRequest("GET", reqURL.String(), nil)
	req.Header.Set("Ocp-Apim-Subscription-Key", subscriptionKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return nil, err
	}
	result := gjson.ParseBytes(body).Get("webPages.value")

	return result.Raw, nil

}

type SearchParams struct {
	Query string `form:"q"`
	Num   int    `form:"num,default=5"`
}
