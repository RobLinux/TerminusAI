package tokenizer

import "strings"

// AnthropicTokenizer implements tokenization for Anthropic Claude models
type AnthropicTokenizer struct {
	BaseTokenizer
}

// NewAnthropicTokenizer creates a new Anthropic tokenizer
func NewAnthropicTokenizer() *AnthropicTokenizer {
	return &AnthropicTokenizer{
		BaseTokenizer: BaseTokenizer{
			tokensPerWord: 1.2, // Anthropic models average ~1.2 tokens per word
		},
	}
}

// GetMaxContextTokens returns the maximum context window for Anthropic models
func (t *AnthropicTokenizer) GetMaxContextTokens(model string) int {
	switch model {
	case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-latest":
		return 200000
	case "claude-3-5-sonnet-20240620":
		return 200000
	case "claude-3-5-haiku-20241022", "claude-3-5-haiku-latest":
		return 200000
	case "claude-3-opus-20240229":
		return 200000
	case "claude-3-sonnet-20240229":
		return 200000
	case "claude-3-haiku-20240307":
		return 200000
	case "claude-3.5-sonnet":
		return 90000
	case "claude-3.7-sonnet", "claude-3.7-sonnet-thought":
		return 200000
	case "claude-sonnet-4":
		return 128000
	default:
		return 90000 // Conservative default (lowest Claude context from the list)
	}
}

// CountTokens provides token estimation for Anthropic models
func (t *AnthropicTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Anthropic's tokenization is similar to OpenAI but with some differences
	charCount := len(text)

	// Anthropic models tend to be slightly more efficient with tokenization
	baseTokens := float64(charCount) / 4.2 // ~4.2 chars per token average

	// Add extra tokens for punctuation
	punctuationBonus := float64(strings.Count(text, ",")+
		strings.Count(text, ".")+
		strings.Count(text, "!")+
		strings.Count(text, "?")+
		strings.Count(text, ";")+
		strings.Count(text, ":")) * 0.3

	// Add tokens for newlines and formatting
	formatBonus := float64(strings.Count(text, "\n")) * 0.8

	totalTokens := int(baseTokens + punctuationBonus + formatBonus)

	// Ensure minimum of 1 token for non-empty text
	if totalTokens == 0 && len(text) > 0 {
		totalTokens = 1
	}

	return totalTokens
}

// EstimateMessageTokens estimates tokens for Anthropic chat messages
func (t *AnthropicTokenizer) EstimateMessageTokens(role, content string) int {
	// Anthropic message overhead is slightly different
	// System messages are separate, user/assistant have minimal overhead
	roleOverhead := 3

	contentTokens := t.CountTokens(content)
	return contentTokens + roleOverhead
}
