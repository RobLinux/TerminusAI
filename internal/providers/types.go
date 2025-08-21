package providers

type ChatMessage struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"`
}

type ChatOptions struct {
	Model string `json:"model,omitempty"`
}

type LLMProvider interface {
	Name() string
	DefaultModel() string
	Chat(messages []ChatMessage, opts *ChatOptions) (string, error)
}