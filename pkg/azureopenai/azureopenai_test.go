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
	"fmt"
	"testing"
)

func TestModels(t *testing.T) {
	type args struct {
		endpoint string
		apikey   string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				endpoint: "https://mirrors2.openai.azure.com",
				apikey:   "696a7729234c438cb38f24da22ee602d",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Models(tt.args.endpoint, tt.args.apikey)
			if err != nil {
				t.Errorf("Models() error = %v", err)
				return
			}
			for _, data := range got.Data {
				fmt.Println(data.Model, data.ID)
			}
		})
	}
}

// curl https://mirrors2.openai.azure.com/openai/deployments?api-version=2023-03-15-preview \
//   -H "Content-Type: application/json" \
//   -H "api-key: 696a7729234c438cb38f24da22ee602d"
