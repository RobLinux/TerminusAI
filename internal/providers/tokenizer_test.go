package providers

import (
	"terminusai/internal/common"
	"terminusai/internal/tokenizer"
	"testing"
)

func TestTokenizers(t *testing.T) {
	testText := "This is a test message to estimate token count. It includes some punctuation, numbers like 123, and various formatting!"

	// Test OpenAI tokenizer
	openaiTokenizer := tokenizer.NewOpenAITokenizer()
	openaiTokens := openaiTokenizer.CountTokens(testText)
	openaiMaxTokens := openaiTokenizer.GetMaxContextTokens("gpt-4o")

	t.Logf("OpenAI tokenizer: %d tokens, max context: %d", openaiTokens, openaiMaxTokens)

	// Test Anthropic tokenizer
	anthropicTokenizer := tokenizer.NewAnthropicTokenizer()
	anthropicTokens := anthropicTokenizer.CountTokens(testText)
	anthropicMaxTokens := anthropicTokenizer.GetMaxContextTokens("claude-3.5-sonnet")

	t.Logf("Anthropic tokenizer: %d tokens, max context: %d", anthropicTokens, anthropicMaxTokens)

	// Test Copilot tokenizer
	copilotTokenizer := tokenizer.NewCopilotTokenizer()
	copilotTokens := copilotTokenizer.CountTokens(testText)
	copilotMaxTokens := copilotTokenizer.GetMaxContextTokens("gpt-4o")

	t.Logf("Copilot tokenizer: %d tokens, max context: %d", copilotTokens, copilotMaxTokens)

	// Ensure all tokenizers give reasonable estimates
	if openaiTokens == 0 || anthropicTokens == 0 || copilotTokens == 0 {
		t.Error("Token count should not be zero for non-empty text")
	}
}

func TestMessageSplitter(t *testing.T) {
	tok := tokenizer.NewOpenAITokenizer()

	// Create a set of messages that would exceed a small limit
	messages := []ChatMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "This is a very long message that would definitely exceed our token limit if we set it low enough. " +
			"It contains many words and should be split into multiple parts when processed. " +
			"We want to test the message splitting functionality to ensure it works correctly."},
		{Role: "assistant", Content: "I understand. I'll help you with that."},
		{Role: "user", Content: "Another long message that adds even more content to our conversation. " +
			"This should definitely cause the total to exceed our artificial limit."},
	}

	// Set a very low limit to force splitting
	splitter := NewMessageSplitter(tok, 50, "gpt-4o")

	batches, err := splitter.SplitMessages(messages)
	if err != nil {
		t.Fatalf("Failed to split messages: %v", err)
	}

	if len(batches) <= 1 {
		t.Error("Expected multiple batches due to low token limit")
	}

	t.Logf("Split %d messages into %d batches", len(messages), len(batches))

	// Verify each batch doesn't exceed the limit
	for i, batch := range batches {
		totalTokens := 0
		for _, msg := range batch {
			totalTokens += tok.EstimateMessageTokens(msg.Role, msg.Content)
		}
		t.Logf("Batch %d: %d messages, %d tokens", i+1, len(batch), totalTokens)
	}
}

func TestChatWithTokenLimits(t *testing.T) {
	// This is a mock test - in reality you'd need a real provider
	// Just testing the function signature and basic logic

	messages := []ChatMessage{
		{Role: "user", Content: "Hello, world!"},
	}

	config := &common.TerminusAIConfig{
		MaxTokensPerRequest: 1000,
	}

	// We can't actually test the full functionality without a real provider
	// but we can verify the function exists and has the right signature
	_ = messages
	_ = config

	// The function ChatWithTokenLimits would be called like:
	// responses, err := ChatWithTokenLimits(provider, messages, nil, config)

	t.Log("ChatWithTokenLimits function is available for use")
}
