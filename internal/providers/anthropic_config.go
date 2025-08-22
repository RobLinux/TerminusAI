package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"terminusai/internal/config"
	"terminusai/internal/tokenizer"
)

// AnthropicProviderConfig is a config-based Anthropic provider
type AnthropicProviderConfig struct {
	name      string
	config    config.ProviderConfig
	cm        *config.ConfigManager
	tokenizer tokenizer.Tokenizer
}

// NewAnthropicProviderWithConfig creates a new Anthropic provider using configuration
func NewAnthropicProviderWithConfig(cm *config.ConfigManager, providerConfig config.ProviderConfig) *AnthropicProviderConfig {
	return &AnthropicProviderConfig{
		name:      "anthropic",
		config:    providerConfig,
		cm:        cm,
		tokenizer: tokenizer.NewAnthropicTokenizer(),
	}
}

func (p *AnthropicProviderConfig) Name() string {
	return p.name
}

func (p *AnthropicProviderConfig) DefaultModel() string {
	if model := p.cm.GetEffectiveModel(); model != "" {
		return model
	}
	return p.config.DefaultModel
}

func (p *AnthropicProviderConfig) GetTokenizer() tokenizer.Tokenizer {
	return p.tokenizer
}

func (p *AnthropicProviderConfig) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	model := p.DefaultModel()
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	// Extract system message
	var system string
	var userMessages []string
	for _, msg := range messages {
		if msg.Role == "system" {
			system = msg.Content
		} else {
			userMessages = append(userMessages, strings.ToUpper(msg.Role)+": "+msg.Content)
		}
	}

	userContent := strings.Join(userMessages, "\n\n")

	reqBody := AnthropicRequest{
		Model:     model,
		System:    system,
		Messages:  []AnthropicMessage{{Role: "user", Content: userContent}},
		MaxTokens: 1024,
	}

	// Handle temperature from configuration
	if temp := p.cm.GetTemperature(); temp != nil {
		reqBody.Temperature = temp
	}

	verbose := p.cm.IsVerbose()
	debug := p.cm.IsDebug()

	if verbose || debug {
		logAnthropicRequest(reqBody, len(messages), debug)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result strings.Builder
	for _, content := range anthropicResp.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}

	text := result.String()
	if verbose || debug {
		logResponse(text, debug)
	}

	return text, nil
}
