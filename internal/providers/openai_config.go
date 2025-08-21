package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terminusai/internal/config"
)

// OpenAIProviderConfig is a config-based OpenAI provider
type OpenAIProviderConfig struct {
	name    string
	config  config.ProviderConfig
	cm      *config.ConfigManager
}

// NewOpenAIProviderWithConfig creates a new OpenAI provider using configuration
func NewOpenAIProviderWithConfig(cm *config.ConfigManager, providerConfig config.ProviderConfig) *OpenAIProviderConfig {
	return &OpenAIProviderConfig{
		name:   "openai",
		config: providerConfig,
		cm:     cm,
	}
}

func (p *OpenAIProviderConfig) Name() string {
	return p.name
}

func (p *OpenAIProviderConfig) DefaultModel() string {
	if model := p.cm.GetEffectiveModel(); model != "" {
		return model
	}
	return p.config.DefaultModel
}

func (p *OpenAIProviderConfig) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	model := p.DefaultModel()
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	reqBody := OpenAIRequest{
		Model:    model,
		Messages: messages,
	}

	// Handle temperature from configuration
	if temp := p.cm.GetTemperature(); temp != nil {
		reqBody.Temperature = temp
	}

	verbose := p.cm.IsVerbose()
	debug := p.cm.IsDebug()

	if verbose || debug {
		logRequest(reqBody, debug)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	result := openaiResp.Choices[0].Message.Content

	if verbose || debug {
		logResponse(result, debug)
	}

	return result, nil
}