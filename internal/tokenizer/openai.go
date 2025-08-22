package tokenizer

import "strings"

// OpenAITokenizer implements tokenization for OpenAI models
type OpenAITokenizer struct {
	BaseTokenizer
}

// NewOpenAITokenizer creates a new OpenAI tokenizer
func NewOpenAITokenizer() *OpenAITokenizer {
	return &OpenAITokenizer{
		BaseTokenizer: BaseTokenizer{
			tokensPerWord: 1.3, // OpenAI models average ~1.3 tokens per word
		},
	}
}

// GetMaxContextTokens returns the maximum context window for OpenAI models
func (t *OpenAITokenizer) GetMaxContextTokens(model string) int {
	switch model {
	case "gpt-4o", "gpt-4o-2024-11-20", "gpt-4o-2024-05-13", "gpt-4o-2024-08-06":
		return 128000
	case "gpt-4-o-preview":
		return 128000
	case "gpt-4o-mini", "gpt-4o-mini-2024-07-18":
		return 128000
	case "gpt-4-0125-preview":
		return 128000
	case "gpt-4", "gpt-4-0613":
		return 32768
	case "gpt-4.1", "gpt-4.1-2025-04-14":
		return 128000
	case "gpt-5":
		return 128000
	case "gpt-3.5-turbo":
		return 16384
	case "gpt-3.5-turbo-0613":
		return 16384
	case "o3-mini", "o3-mini-2025-01-31", "o3-mini-paygo":
		return 200000
	case "o4-mini":
		return 128000
	case "o4-mini-2025-04-16":
		return 200000
	default:
		return 16384 // Conservative default (lowest OpenAI context)
	}
}

// CountTokens provides a more accurate estimation for OpenAI models
func (t *OpenAITokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// More sophisticated token counting for OpenAI
	// This is still an approximation, but better than simple word counting

	// Count characters and apply a rough conversion factor
	charCount := len(text)

	// OpenAI tokenization roughly follows these patterns:
	// - Average English word is ~4-5 characters
	// - Punctuation and spaces count as separate tokens
	// - Special characters may be multiple tokens

	// Estimate based on character count with adjustments
	baseTokens := float64(charCount) / 4.0 // ~4 chars per token average

	// Add extra tokens for punctuation and special formatting
	punctuationBonus := float64(strings.Count(text, ",")+
		strings.Count(text, ".")+
		strings.Count(text, "!")+
		strings.Count(text, "?")+
		strings.Count(text, ";")+
		strings.Count(text, ":")) * 0.5

	// Add tokens for newlines and formatting
	formatBonus := float64(strings.Count(text, "\n")) * 1.0

	totalTokens := int(baseTokens + punctuationBonus + formatBonus)

	// Ensure minimum of 1 token for non-empty text
	if totalTokens == 0 && len(text) > 0 {
		totalTokens = 1
	}

	return totalTokens
}

// EstimateMessageTokens estimates tokens for OpenAI chat messages
func (t *OpenAITokenizer) EstimateMessageTokens(role, content string) int {
	// OpenAI chat messages have specific overhead:
	// - Every message follows <|start|>{role/name}\n{content}<|end|>\n
	// - Role tokens: ~3-4 tokens per message
	roleOverhead := 4

	contentTokens := t.CountTokens(content)
	return contentTokens + roleOverhead
}
