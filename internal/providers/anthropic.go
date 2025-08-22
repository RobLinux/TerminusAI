package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"terminusai/internal/common"
	"terminusai/internal/tokenizer"
)

type AnthropicProvider struct {
	name          string
	defaultModel  string
	modelOverride string
	apiKey        string
	config        *common.TerminusAIConfig
	tokenizer     tokenizer.Tokenizer
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
}

type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
}

func NewAnthropicProvider(modelOverride string) *AnthropicProvider {
	defaultModel := "claude-3-5-sonnet-latest"

	return &AnthropicProvider{
		name:          "anthropic",
		defaultModel:  defaultModel,
		modelOverride: modelOverride,
		apiKey:        "", // Legacy provider - should use config-based provider instead
		config:        nil,
		tokenizer:     tokenizer.NewAnthropicTokenizer(),
	}
}

func (p *AnthropicProvider) Name() string {
	return p.name
}

func (p *AnthropicProvider) DefaultModel() string {
	return p.defaultModel
}

func (p *AnthropicProvider) GetTokenizer() tokenizer.Tokenizer {
	return p.tokenizer
}

func (p *AnthropicProvider) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	model := p.defaultModel
	if opts != nil && opts.Model != "" {
		model = opts.Model
	} else if p.modelOverride != "" {
		model = p.modelOverride
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

	// Legacy provider uses default temperature handling from options only

	verbose := false // Legacy mode - no logging
	debug := false   // Legacy mode - no logging

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
	req.Header.Set("x-api-key", p.apiKey)
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

func logAnthropicRequest(reqBody AnthropicRequest, msgCount int, debug bool) {
	var preview string
	if debug {
		previewJSON, _ := json.Marshal(reqBody)
		preview = string(previewJSON)
	} else {
		preview = fmt.Sprintf(`{"model":"%s","system":"...","messages":"[%d msg]","max_tokens":%d}`,
			reqBody.Model, msgCount, reqBody.MaxTokens)
	}
	fmt.Printf("[http] Anthropic POST /messages model=%s body=%s\n", reqBody.Model, preview)
}
