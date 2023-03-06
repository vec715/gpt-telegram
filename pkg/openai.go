package pkg

import (
	"context"
	"fmt"

	gogpt "github.com/sashabaranov/go-gpt3"
)

type OpenAI struct {
	client *gogpt.Client
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{gogpt.NewClient(apiKey)}
}

func (o *OpenAI) CreateCompletionRequest(user string, messages []gogpt.ChatCompletionMessage) (string, error) {
	// insert the system prompt at the 0th index of the messages slice at each request
	prompt := o.AddSystemPrompt(messages)
	resp, err := o.client.CreateChatCompletion(context.Background(), gogpt.ChatCompletionRequest{
		Model:       gogpt.GPT3Dot5Turbo,
		Temperature: 0.9,
		User:        user,
		Messages:    prompt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create completion request: %w", err)
	}

	return resp.Choices[len(resp.Choices)-1].Message.Content, nil
}

// SystemPrompt returns the default system prompt
func (s *OpenAI) SystemPrompt() gogpt.ChatCompletionMessage {
	return gogpt.ChatCompletionMessage{
		Role:    "system",
		Content: "You are a useful chat assistant. Answers should be as detailed as possible using markdown formatting. Answers should consist of the following blocks (without explicitly mentioning the block name):\n1. Description of the topic/problem/question/object\n2. Detailed explanation of the answer to the problem\n3. Examples\n4. Criticism/Another point of view\n5. A summary\n\nRemember that you are an intelligent assistant, so all your answers must be scientifically validated, as if you were an expert in your field.",
	}
}

// AddSystemPrompt adds the system prompt to the beginning of the messages slice
func (s *OpenAI) AddSystemPrompt(messages []gogpt.ChatCompletionMessage) []gogpt.ChatCompletionMessage {
	return append([]gogpt.ChatCompletionMessage{s.SystemPrompt()}, messages...)
}
