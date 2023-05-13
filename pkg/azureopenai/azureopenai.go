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

var (
	ENDPOINT        string
	API_KEY         string
	DEPLOYMENT_NAME string
)
