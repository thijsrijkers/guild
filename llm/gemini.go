package llm

import (
	"context"
	"errors"
	"os"

	"google.golang.org/genai"
)

type GeminiClient struct {
	model string
}

func NewGeminiClient(model string) (LLM, error) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiClient{model: model}, nil
}

func (g *GeminiClient) Ask(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}

	result, err := client.Models.GenerateContent(ctx, g.model,
		genai.Text(prompt), nil,
	)
	if err != nil {
		return "", err
	}

	if result == nil || result.Text() == "" {
		return "", errors.New("empty response from Gemini")
	}

	return result.Text(), nil
}
