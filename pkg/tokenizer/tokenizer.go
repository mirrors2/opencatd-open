package tokenizer

import (
	"fmt"
	"log"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
)

func NumTokensFromMessages(messages []openai.ChatCompletionMessage, model string) (numTokens int) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		err = fmt.Errorf("EncodingForModel: %v", err)
		log.Println(err)
		return
	}

	var tokensPerMessage, tokensPerName int

	switch model {
	case "gpt-3.5-turbo",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0314",
		"gpt-4-32k-0613":
		tokensPerMessage = 3
		tokensPerName = 1
	case "gpt-3.5-turbo-0301":
		tokensPerMessage = 4 // every message follows <|start|>{role/name}\n{content}<|end|>\n
		tokensPerName = -1   // if there's a name, the role is omitted
	default:
		if strings.Contains(model, "gpt-3.5-turbo") {
			log.Println("warning: gpt-3.5-turbo may update over time. Returning num tokens assuming gpt-3.5-turbo-0613.")
			return NumTokensFromMessages(messages, "gpt-3.5-turbo-0613")
		} else if strings.Contains(model, "gpt-4") {
			log.Println("warning: gpt-4 may update over time. Returning num tokens assuming gpt-4-0613.")
			return NumTokensFromMessages(messages, "gpt-4-0613")
		} else {
			err = fmt.Errorf("warning: unknown model [%s]. Use default calculation method converted tokens.", model)
			log.Println(err)
			return NumTokensFromMessages(messages, "gpt-3.5-turbo-0613")
		}
	}

	for _, message := range messages {
		numTokens += tokensPerMessage
		numTokens += len(tkm.Encode(message.Content, nil, nil))
		numTokens += len(tkm.Encode(message.Role, nil, nil))
		numTokens += len(tkm.Encode(message.Name, nil, nil))
		if message.Name != "" {
			numTokens += tokensPerName
		}
	}
	numTokens += 3
	return numTokens
}

func NumTokensFromStr(messages string, model string) (num_tokens int) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Unsupport Model,use cl100k_base Encode")
		tkm, _ = tiktoken.GetEncoding("cl100k_base")
	}

	num_tokens += len(tkm.Encode(messages, nil, nil))
	return num_tokens
}

// https://openai.com/pricing
func Cost(model string, promptCount, completionCount int) float64 {
	var cost, prompt, completion float64
	prompt = float64(promptCount)
	completion = float64(completionCount)

	switch model {
	case "gpt-3.5-turbo-0301":
		cost = 0.002 * float64((prompt+completion)/1000)
	case "gpt-3.5-turbo", "gpt-3.5-turbo-0613", "gpt-3.5-turbo-1106", "gpt-3.5-turbo-0125":
		cost = 0.0015*float64((prompt)/1000) + 0.002*float64(completion/1000)
	case "gpt-3.5-turbo-16k", "gpt-3.5-turbo-16k-0613":
		cost = 0.003*float64((prompt)/1000) + 0.004*float64(completion/1000)
	case "gpt-4", "gpt-4-0613", "gpt-4-0314":
		cost = 0.03*float64(prompt/1000) + 0.06*float64(completion/1000)
	case "gpt-4-32k", "gpt-4-32k-0314", "gpt-4-32k-0613":
		cost = 0.06*float64(prompt/1000) + 0.12*float64(completion/1000)
	case "gpt-4-1106-preview", "gpt-4-vision-preview", "gpt-4-0125-preview", "gpt-4-turbo-preview":
		cost = 0.01*float64(prompt/1000) + 0.03*float64(completion/1000)
	case "gpt-4-turbo", "gpt-4-turbo-2024-04-09":
		cost = 0.01*float64(prompt/1000) + 0.03*float64(completion/1000)
	case "gpt-4o", "gpt-4o-2024-05-13":
		cost = 0.005*float64(prompt/1000) + 0.015*float64(completion/1000)
	case "gpt-4o-mini", "gpt-4o-mini-2024-07-18":
		cost = 0.00015*float64(prompt/1000) + 0.0006*float64(completion/1000)
	case "whisper-1":
		// 0.006$/min
		cost = 0.006 * float64(prompt+completion) / 60
	case "tts-1":
		cost = 0.015 * float64(prompt+completion)
	case "tts-1-hd":
		cost = 0.03 * float64(prompt+completion)
	case "dall-e-2.256x256":
		cost = float64(0.016 * completion)
	case "dall-e-2.512x512":
		cost = float64(0.018 * completion)
	case "dall-e-2.1024x1024":
		cost = float64(0.02 * completion)
	case "dall-e-3.256x256":
		cost = float64(0.04 * completion)
	case "dall-e-3.512x512":
		cost = float64(0.04 * completion)
	case "dall-e-3.1024x1024":
		cost = float64(0.04 * completion)
	case "dall-e-3.1024x1792", "dall-e-3.1792x1024":
		cost = float64(0.08 * completion)
	case "dall-e-3.256x256.hd":
		cost = float64(0.08 * completion)
	case "dall-e-3.512x512.hd":
		cost = float64(0.08 * completion)
	case "dall-e-3.1024x1024.hd":
		cost = float64(0.08 * completion)
	case "dall-e-3.1024x1792.hd", "dall-e-3.1792x1024.hd":
		cost = float64(0.12 * completion)

	// claude /million tokens
	// https://aws.amazon.com/cn/bedrock/pricing/
	case "claude-v1", "claude-v1-100k":
		cost = 11.02/1000000*float64(prompt) + (32.68/1000000)*float64(completion)
	case "claude-instant-v1", "claude-instant-v1-100k":
		cost = (1.63/1000000)*float64(prompt) + (5.51/1000000)*float64(completion)
	case "claude-2", "claude-2.1":
		cost = (8.0/1000000)*float64(prompt) + (24.0/1000000)*float64(completion)
	case "claude-3-haiku":
		cost = (0.00025/1000)*float64(prompt) + (0.00125/1000)*float64(completion)
	case "claude-3-sonnet":
		cost = (0.003/1000)*float64(prompt) + (0.015/1000)*float64(completion)
	case "claude-3-opus":
		cost = (0.015/1000)*float64(prompt) + (0.075/1000)*float64(completion)
	case "claude-3-haiku-20240307":
		cost = (0.00025/1000)*float64(prompt) + (0.00125/1000)*float64(completion)
	case "claude-3-sonnet-20240229":
		cost = (0.003/1000)*float64(prompt) + (0.015/1000)*float64(completion)
	case "claude-3-opus-20240229":
		cost = (0.015/1000)*float64(prompt) + (0.075/1000)*float64(completion)

	// google
	// https://ai.google.dev/pricing?hl=zh-cn
	case "gemini-pro":
		cost = (0.0005/1000)*float64(prompt) + (0.0015/1000)*float64(completion)
	case "gemini-pro-vision":
		cost = (0.0005/1000)*float64(prompt) + (0.0015/1000)*float64(completion)
	case "gemini-1.5-pro-latest":
		cost = (0.0035/1000)*float64(prompt) + (0.0105/1000)*float64(completion)
	case "gemini-1.5-flash-latest":
		cost = (0.00035/1000)*float64(prompt) + (0.00053/1000)*float64(completion)

	// Mistral AI
	// https://docs.mistral.ai/platform/pricing/
	case "mistral-small-latest":
		cost = (0.002/1000)*float64(prompt) + (0.006/1000)*float64(completion)
	case "mistral-medium-latest":
		cost = (0.0027/1000)*float64(prompt) + (0.0081/1000)*float64(completion)
	case "mistral-large-latest":
		cost = (0.008/1000)*float64(prompt) + (0.024/1000)*float64(completion)

	default:
		if strings.Contains(model, "gpt-3.5-turbo") {
			cost = 0.003 * float64((prompt+completion)/1000)
		} else if strings.Contains(model, "gpt-4") {
			cost = 0.06 * float64((prompt+completion)/1000)
		} else {
			cost = 0.002 * float64((prompt+completion)/1000)
		}
	}
	return cost
}
