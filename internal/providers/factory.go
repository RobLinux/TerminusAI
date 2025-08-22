package providers

import (
	"fmt"
	"strings"

	"terminusai/internal/config"
)

func GetProvider(name string, model string) (LLMProvider, error) {
	cm := config.GetConfigManager()
	
	providerName := strings.ToLower(name)
	
	// Handle provider name variations
	switch providerName {
	case "claude":
		providerName = "anthropic"
	case "copilot-api", "copilot-completion":
		providerName = "copilot"
	}
	
	// Try config-based approach first
	if provider, err := NewProviderWithConfig(cm, providerName); err == nil {
		return provider, nil
	}
	
	// If config doesn't exist, create basic provider config and use config-based constructors
	cfg, _ := config.LoadConfig() // Load config, ignore errors for fallback
	
	// Create a basic provider config
	providerConfig := config.ProviderConfig{
		Enabled:      true,
		DefaultModel: model,
	}
	
	// Set API key from loaded config if available
	if cfg != nil {
		switch providerName {
		case "openai":
			providerConfig.APIKey = cfg.OpenAIAPIKey
		case "anthropic":
			providerConfig.APIKey = cfg.AnthropicAPIKey
		case "copilot":
			providerConfig.APIKey = cfg.GitHubToken
		}
	}
	
	switch providerName {
	case "openai":
		return NewOpenAIProviderWithConfig(cm, providerConfig), nil
	case "anthropic":
		return NewAnthropicProviderWithConfig(cm, providerConfig), nil
	case "copilot":
		return NewCopilotProviderWithConfig(cm, providerConfig), nil
	default:
		return nil, fmt.Errorf("unknown provider '%s'. Use openai|anthropic|copilot", name)
	}
}
