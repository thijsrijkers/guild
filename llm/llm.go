package llm

import (
	"context"
	"os"
)

type LLM interface {
	Ask(ctx context.Context, prompt string) (string, error)
}

func NewFromEnv() (LLM, error) {
	provider := os.Getenv("LLM_PROVIDER")
	model := os.Getenv("LLM_MODEL")

	switch provider {
	case "ollama":
		if model == "" {
			model = "deepseek-coder"
		}
		return NewOllamaClient(model), nil
	case "gemini":
    return NewGeminiClient(model)
	case "claude":
    return NewClaudeClient(model)
	case "openai":
    return NewOpenAIClient(model)
	default:
		return NewOllamaClient("deepseek-coder"), nil
	}
}
