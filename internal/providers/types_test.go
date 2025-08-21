package providers

import (
	"testing"
)

func TestChatMessage(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello, world!",
	}

	if msg.Role != "user" {
		t.Errorf("Expected Role to be 'user', got %q", msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("Expected Content to be 'Hello, world!', got %q", msg.Content)
	}
}

func TestChatOptions(t *testing.T) {
	opts := ChatOptions{
		Model: "gpt-4",
	}

	if opts.Model != "gpt-4" {
		t.Errorf("Expected Model to be 'gpt-4', got %q", opts.Model)
	}
}

func TestChatOptionsEmpty(t *testing.T) {
	opts := ChatOptions{}

	if opts.Model != "" {
		t.Errorf("Expected empty Model, got %q", opts.Model)
	}
}

// MockLLMProvider for testing
type MockLLMProvider struct {
	name         string
	defaultModel string
	response     string
	err          error
}

func (m *MockLLMProvider) Name() string {
	return m.name
}

func (m *MockLLMProvider) DefaultModel() string {
	return m.defaultModel
}

func (m *MockLLMProvider) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestLLMProviderInterface(t *testing.T) {
	provider := &MockLLMProvider{
		name:         "test-provider",
		defaultModel: "test-model",
		response:     "test response",
	}

	if provider.Name() != "test-provider" {
		t.Errorf("Expected Name to be 'test-provider', got %q", provider.Name())
	}
	if provider.DefaultModel() != "test-model" {
		t.Errorf("Expected DefaultModel to be 'test-model', got %q", provider.DefaultModel())
	}

	messages := []ChatMessage{
		{Role: "user", Content: "test message"},
	}

	response, err := provider.Chat(messages, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if response != "test response" {
		t.Errorf("Expected response to be 'test response', got %q", response)
	}
}
