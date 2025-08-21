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
	case "github":
		return NewGitHubProvider(model), nil
	case "copilot", "copilot-api", "copilot-completion":
		// For explicit Copilot provider, enable Copilot mode
		return NewGitHubProviderWithCopilotMode(model), nil
	default:
		return nil, errors.New("unknown provider '" + name + "'. Use openai|anthropic|github|copilot")
	}
}
