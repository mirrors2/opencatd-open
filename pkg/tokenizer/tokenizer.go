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
	case "gpt-3.5-turbo", "gpt-3.5-turbo-0613":
		cost = 0.0015*float64((prompt)/1000) + 0.002*float64(completion/1000)
	case "gpt-3.5-turbo-16k", "gpt-3.5-turbo-16k-0613":
		cost = 0.003*float64((prompt)/1000) + 0.004*float64(completion/1000)
	case "gpt-4", "gpt-4-0613", "gpt-4-0314":
		cost = 0.03*float64(prompt/1000) + 0.06*float64(completion/1000)
	case "gpt-4-32k", "gpt-4-32k-0613":
		cost = 0.06*float64(prompt/1000) + 0.12*float64(completion/1000)
	case "whisper-1":
		// 0.006$/min
		cost = 0.006 * float64(prompt+completion) / 60
	// claude /million tokens
	case "claude-v1", "claude-v1-100k":
		cost = 11.02/1000000*float64(prompt) + (32.68/1000000)*float64(completion)
	case "claude-instant-v1", "claude-instant-v1-100k":
		cost = (1.63/1000000)*float64(prompt) + (5.51/1000000)*float64(completion)
	case "claude-2":
		cost = (11.02/1000000)*float64(prompt) + (32.68/1000000)*float64(completion)
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
