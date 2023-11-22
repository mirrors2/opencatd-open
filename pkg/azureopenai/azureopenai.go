/*
https://learn.microsoft.com/zh-cn/azure/cognitive-services/openai/chatgpt-quickstart
https://learn.microsoft.com/zh-cn/azure/ai-services/openai/reference#chat-completions

curl $AZURE_OPENAI_ENDPOINT/openai/deployments/gpt-35-turbo/chat/completions?api-version=2023-03-15-preview \
  -H "Content-Type: application/json" \
  -H "api-key: $AZURE_OPENAI_KEY" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "你好"}]
  }'

https://learn.microsoft.com/zh-cn/rest/api/cognitiveservices/azureopenaistable/models/list?tabs=HTTP

  curl $AZURE_OPENAI_ENDPOINT/openai/deployments?api-version=2022-12-01 \
  -H "Content-Type: application/json" \
  -H "api-key: $AZURE_OPENAI_KEY" \

> GPT-4 Turbo
https://techcommunity.microsoft.com/t5/ai-azure-ai-services-blog/azure-openai-service-launches-gpt-4-turbo-and-gpt-3-5-turbo-1106/ba-p/3985962

*/

package azureopenai

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

var (
	ENDPOINT        string
	API_KEY         string
	DEPLOYMENT_NAME string
)

type ModelsList struct {
	Data []struct {
		ScaleSettings struct {
			ScaleType string `json:"scale_type"`
		} `json:"scale_settings"`
		Model     string `json:"model"`
		Owner     string `json:"owner"`
		ID        string `json:"id"`
		Status    string `json:"status"`
		CreatedAt int    `json:"created_at"`
		UpdatedAt int    `json:"updated_at"`
		Object    string `json:"object"`
	} `json:"data"`
	Object string `json:"object"`
}

func Models(endpoint, apikey string) (*ModelsList, error) {
	endpoint = RemoveTrailingSlash(endpoint)
	var modelsl ModelsList
	req, _ := http.NewRequest(http.MethodGet, endpoint+"/openai/deployments?api-version=2022-12-01", nil)
	req.Header.Set("api-key", apikey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&modelsl)
	if err != nil {
		return nil, err
	}
	return &modelsl, nil

}

func RemoveTrailingSlash(s string) string {
	const prefix = "openai.azure.com/"
	if strings.HasSuffix(strings.TrimSpace(s), prefix) && strings.HasSuffix(s, "/") {
		return s[:len(s)-1]
	}
	return s
}

func GetResourceName(url string) string {
	re := regexp.MustCompile(`https?://(.+)\.openai\.azure\.com/?`)
	match := re.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}
