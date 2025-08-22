package providers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"terminusai/internal/common"
	"terminusai/internal/config"
	"terminusai/internal/tokenizer"
)

// CopilotProviderConfig is a config-based Copilot provider
type CopilotProviderConfig struct {
	name      string
	config    config.ProviderConfig
	cm        *config.ConfigManager
	tokenizer tokenizer.Tokenizer
}

// NewCopilotProviderWithConfig creates a new Copilot provider using configuration
func NewCopilotProviderWithConfig(cm *config.ConfigManager, providerConfig config.ProviderConfig) *CopilotProviderConfig {
	providerName := "copilot"

	return &CopilotProviderConfig{
		name:      providerName,
		config:    providerConfig,
		cm:        cm,
		tokenizer: tokenizer.NewCopilotTokenizer(),
	}
}

func (p *CopilotProviderConfig) Name() string {
	return p.name
}

func (p *CopilotProviderConfig) DefaultModel() string {
	if model := p.cm.GetEffectiveModel(); model != "" {
		return model
	}
	return p.config.DefaultModel
}

func (p *CopilotProviderConfig) GetTokenizer() tokenizer.Tokenizer {
	return p.tokenizer
}

func (p *CopilotProviderConfig) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	// If in Copilot mode, delegate to standalone provider for now
	if p.name == "copilot" {
		// Get the effective model from configuration
		model := p.DefaultModel()
		if opts != nil && opts.Model != "" {
			model = opts.Model
		}

		// Create a standalone Copilot provider in Copilot mode with the correct model
		standalone := NewCopilotProvider(model)
		// Pass config for access to GitHub token
		cfg := &common.TerminusAIConfig{
			GitHubToken: p.config.APIKey,
			Model:       model,
		}
		standalone.config = cfg
		return standalone.ChatWithConfig(messages, opts, cfg)
	}

	// Standard Copilot Models chat
	model := p.DefaultModel()
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	url := p.config.BaseURL + "/v1/chat/completions"

	reqBody := CopilotRequest{
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
		logCopilotRequest(reqBody, len(messages), p.config.APIKey, url, debug)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Create client with insecure TLS (matching original behavior)
	// Note: This is insecure and should be used cautiously
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

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
		return "", p.handleError(resp.StatusCode, body)
	}

	var copilotResp CopilotResponse
	if err := json.Unmarshal(body, &copilotResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(copilotResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	result := copilotResp.Choices[0].Message.Content

	if verbose || debug {
		logResponse(result, debug)
	}

	return result, nil
}

func (p *CopilotProviderConfig) handleError(statusCode int, body []byte) error {
	details := string(body)

	var errorResp CopilotErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil {
		details = fmt.Sprintf(`{"error":{"message":"%s"},"message":"%s"}`, errorResp.Error.Message, errorResp.Message)

		msg := errorResp.Error.Message
		if msg == "" {
			msg = errorResp.Message
		}

		if statusCode == 401 || statusCode == 403 {
			if strings.Contains(strings.ToLower(msg), "models is disabled") {
				return fmt.Errorf("Copilot Models appears disabled for your org. Choose another provider (run: terminusai setup) or override: --provider openai|anthropic")
			}
			if strings.Contains(strings.ToLower(msg), "unauthoriz") ||
				strings.Contains(strings.ToLower(msg), "invalid") ||
				strings.Contains(strings.ToLower(msg), "token") {
				return fmt.Errorf("Copilot token unauthorized. Re-run setup to authenticate or paste a valid Copilot/Models token")
			}
		}
	}

	return fmt.Errorf("Copilot provider error: %d %s", statusCode, details)
}
