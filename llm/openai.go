package llm

import (
	"context"
	"errors"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	model string
}

func NewOpenAIClient(model string) (LLM, error) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIClient{model: model}, nil
}

func (o *OpenAIClient) Ask(ctx context.Context, prompt string) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	resp, err := client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model: o.model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("empty response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
