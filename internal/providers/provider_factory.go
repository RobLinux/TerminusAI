package providers

import (
	"fmt"

	"terminusai/internal/config"
)

// NewProviderWithConfig creates a provider using the configuration manager
func NewProviderWithConfig(cm *config.ConfigManager, providerName string) (LLMProvider, error) {
	providerConfig, exists := cm.GetProviderConfig(providerName)
	if !exists {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	if !providerConfig.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", providerName)
	}

	switch providerName {
	case "openai":
		return NewOpenAIProviderWithConfig(cm, providerConfig), nil
	case "anthropic":
		return NewAnthropicProviderWithConfig(cm, providerConfig), nil
	case "copilot", "copilot-api":
		return NewCopilotProviderWithConfig(cm, providerConfig), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// GetProviderWithFallback attempts to create a provider with config, falls back to legacy method
func GetProviderWithFallback(providerName, modelOverride string) (LLMProvider, error) {
	cm := config.GetConfigManager()

	// Try new config-based approach first
	if provider, err := NewProviderWithConfig(cm, providerName); err == nil {
		return provider, nil
	}

	// Fall back to legacy method for backward compatibility
	return GetProvider(providerName, modelOverride)
}
