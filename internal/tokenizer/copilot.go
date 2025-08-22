package tokenizer

import "strings"

// CopilotTokenizer implements tokenization for GitHub Copilot models
type CopilotTokenizer struct {
	BaseTokenizer
}

// NewCopilotTokenizer creates a new Copilot tokenizer
func NewCopilotTokenizer() *CopilotTokenizer {
	return &CopilotTokenizer{
		BaseTokenizer: BaseTokenizer{
			tokensPerWord: 1.3, // GitHub Copilot uses OpenAI-like tokenization
		},
	}
}

// GetMaxContextTokens returns the maximum context window for Copilot models
func (t *CopilotTokenizer) GetMaxContextTokens(model string) int {
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
	case "claude-3.5-sonnet":
		return 90000
	case "claude-3.7-sonnet", "claude-3.7-sonnet-thought":
		return 200000
	case "claude-sonnet-4":
		return 128000
	case "gemini-2.0-flash-001":
		return 1000000
	case "gemini-2.5-pro":
		return 128000
	default:
		return 16384 // Conservative default (lowest context from the list)
	}
}

// CountTokens provides token estimation for Copilot models
func (t *CopilotTokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// GitHub Copilot uses OpenAI-style tokenization
	charCount := len(text)

	// Similar to OpenAI tokenization
	baseTokens := float64(charCount) / 4.0 // ~4 chars per token average

	// Add extra tokens for punctuation
	punctuationBonus := float64(strings.Count(text, ",")+
		strings.Count(text, ".")+
		strings.Count(text, "!")+
		strings.Count(text, "?")+
		strings.Count(text, ";")+
		strings.Count(text, ":")) * 0.5

	// Add tokens for newlines and formatting
	formatBonus := float64(strings.Count(text, "\n")) * 1.0

	// Code-specific adjustments (Copilot is primarily for code)
	codeBonus := float64(strings.Count(text, "{")+
		strings.Count(text, "}")+
		strings.Count(text, "(")+
		strings.Count(text, ")")+
		strings.Count(text, "[")+
		strings.Count(text, "]")) * 0.2

	totalTokens := int(baseTokens + punctuationBonus + formatBonus + codeBonus)

	// Ensure minimum of 1 token for non-empty text
	if totalTokens == 0 && len(text) > 0 {
		totalTokens = 1
	}

	return totalTokens
}

// EstimateMessageTokens estimates tokens for Copilot chat messages
func (t *CopilotTokenizer) EstimateMessageTokens(role, content string) int {
	// Copilot message overhead similar to OpenAI
	roleOverhead := 4

	contentTokens := t.CountTokens(content)
	return contentTokens + roleOverhead
}
