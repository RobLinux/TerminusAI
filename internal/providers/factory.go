package providers

import (
	"errors"
	"strings"
)

func GetProvider(name string, model string) (LLMProvider, error) {
	switch strings.ToLower(name) {
	case "openai":
		return NewOpenAIProvider(model), nil
	case "anthropic", "claude":
		return NewAnthropicProvider(model), nil
	case "github", "copilot":
		return NewGitHubProvider(model), nil
	default:
		return nil, errors.New("unknown provider '" + name + "'. Use openai|anthropic|github")
	}
}