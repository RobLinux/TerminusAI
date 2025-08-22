package providers

import (
	"fmt"
	"terminusai/internal/common"
)

// ChatWithTokenLimits handles chat requests with token limit enforcement and automatic splitting
func ChatWithTokenLimits(provider LLMProvider, messages []ChatMessage, opts *ChatOptions, config *common.TerminusAIConfig) ([]string, error) {
	if config == nil {
		// No config, use provider defaults
		response, err := provider.Chat(messages, opts)
		if err != nil {
			return nil, err
		}
		return []string{response}, nil
	}

	tokenizer := provider.GetTokenizer()
	model := provider.DefaultModel()
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	// Determine max tokens per request
	maxTokens := config.MaxTokensPerRequest
	if maxTokens <= 0 {
		// Use model's max context window
		maxTokens = tokenizer.GetMaxContextTokens(model)
	}

	// Create message splitter
	splitter := NewMessageSplitter(tokenizer, maxTokens, model)

	// Split messages if necessary
	messageBatches, err := splitter.SplitMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to split messages: %w", err)
	}

	// Process each batch
	var responses []string
	for i, batch := range messageBatches {
		response, err := provider.Chat(batch, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to process batch %d: %w", i+1, err)
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// EstimateTokensForMessages estimates total tokens for a conversation
func EstimateTokensForMessages(provider LLMProvider, messages []ChatMessage) int {
	tokenizer := provider.GetTokenizer()
	total := 0

	for _, msg := range messages {
		total += tokenizer.EstimateMessageTokens(msg.Role, msg.Content)
	}

	return total
}

// CheckTokenLimits checks if messages exceed the configured or model limits
func CheckTokenLimits(provider LLMProvider, messages []ChatMessage, config *common.TerminusAIConfig, model string) (bool, int, int) {
	tokenizer := provider.GetTokenizer()
	totalTokens := EstimateTokensForMessages(provider, messages)

	// Determine the effective limit
	var effectiveLimit int
	if config != nil && config.MaxTokensPerRequest > 0 {
		effectiveLimit = config.MaxTokensPerRequest
	} else {
		effectiveLimit = tokenizer.GetMaxContextTokens(model)
	}

	exceedsLimit := totalTokens > effectiveLimit
	return exceedsLimit, totalTokens, effectiveLimit
}
