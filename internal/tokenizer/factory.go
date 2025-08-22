package tokenizer

import (
	"fmt"
	"terminusai/internal/common"
)

// TokenizerFactory creates tokenizers for different providers
type TokenizerFactory struct{}

// NewTokenizerFactory creates a new tokenizer factory
func NewTokenizerFactory() *TokenizerFactory {
	return &TokenizerFactory{}
}

// CreateTokenizer creates a tokenizer for the specified provider
func (f *TokenizerFactory) CreateTokenizer(provider string) (Tokenizer, error) {
	switch provider {
	case common.ProviderOpenAI:
		return NewOpenAITokenizer(), nil
	case common.ProviderAnthropic:
		return NewAnthropicTokenizer(), nil
	case common.ProviderCopilot:
		return NewCopilotTokenizer(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// GetTokenizer is a convenience function to get a tokenizer for a provider
func GetTokenizer(provider string) (Tokenizer, error) {
	factory := NewTokenizerFactory()
	return factory.CreateTokenizer(provider)
}
