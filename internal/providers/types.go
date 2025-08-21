package providers

type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

type ChatOptions struct {
	Model string `json:"model,omitempty"`
}

type CompletionOptions struct {
	Language    string `json:"language,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
	Temperature int    `json:"temperature,omitempty"`
	Suffix      string `json:"suffix,omitempty"`
}

type LLMProvider interface {
	Name() string
	DefaultModel() string
	Chat(messages []ChatMessage, opts *ChatOptions) (string, error)
}

type CompletionProvider interface {
	LLMProvider
	Complete(prompt string, opts *CompletionOptions) (string, error)
}

// Ensure GitHubProvider implements both interfaces
var _ LLMProvider = (*GitHubProvider)(nil)
var _ CompletionProvider = (*GitHubProvider)(nil)
