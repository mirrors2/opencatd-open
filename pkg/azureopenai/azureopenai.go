/*
https://learn.microsoft.com/zh-cn/azure/cognitive-services/openai/chatgpt-quickstart

curl $AZURE_OPENAI_ENDPOINT/openai/deployments/gpt-35-turbo/chat/completions?api-version=2023-03-15-preview \
  -H "Content-Type: application/json" \
  -H "api-key: $AZURE_OPENAI_KEY" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "你好"}]
  }'

*/

package azureopenai

import (
	"encoding/json"
	"net/http"
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
	endpoint = removeTrailingSlash(endpoint)
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

func removeTrailingSlash(s string) string {
	const prefix = "openai.azure.com/"
	if strings.HasPrefix(s, prefix) && strings.HasSuffix(s, "/") {
		return s[:len(s)-1]
	}
	return s
}
