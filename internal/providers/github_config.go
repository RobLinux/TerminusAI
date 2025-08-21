package providers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"terminusai/internal/config"
)

// GitHubProviderConfig is a config-based GitHub provider
type GitHubProviderConfig struct {
	name   string
	config config.ProviderConfig
	cm     *config.ConfigManager
}

// NewGitHubProviderWithConfig creates a new GitHub provider using configuration
func NewGitHubProviderWithConfig(cm *config.ConfigManager, providerConfig config.ProviderConfig) *GitHubProviderConfig {
	return &GitHubProviderConfig{
		name:   "github",
		config: providerConfig,
		cm:     cm,
	}
}

func (p *GitHubProviderConfig) Name() string {
	return p.name
}

func (p *GitHubProviderConfig) DefaultModel() string {
	if model := p.cm.GetEffectiveModel(); model != "" {
		return model
	}
	return p.config.DefaultModel
}

func (p *GitHubProviderConfig) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	model := p.DefaultModel()
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	url := p.config.BaseURL + "/v1/chat/completions"

	reqBody := GitHubRequest{
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
		logGitHubRequest(reqBody, len(messages), p.config.APIKey, url, debug)
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

	var githubResp GitHubResponse
	if err := json.Unmarshal(body, &githubResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(githubResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	result := githubResp.Choices[0].Message.Content

	if verbose || debug {
		logResponse(result, debug)
	}

	return result, nil
}

func (p *GitHubProviderConfig) handleError(statusCode int, body []byte) error {
	details := string(body)
	
	var errorResp GitHubErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil {
		details = fmt.Sprintf(`{"error":{"message":"%s"},"message":"%s"}`, errorResp.Error.Message, errorResp.Message)
		
		msg := errorResp.Error.Message
		if msg == "" {
			msg = errorResp.Message
		}
		
		if statusCode == 401 || statusCode == 403 {
			if strings.Contains(strings.ToLower(msg), "models is disabled") {
				return fmt.Errorf("GitHub Models appears disabled for your org. Choose another provider (run: terminusai setup) or override: --provider openai|anthropic")
			}
			if strings.Contains(strings.ToLower(msg), "unauthoriz") || 
			   strings.Contains(strings.ToLower(msg), "invalid") || 
			   strings.Contains(strings.ToLower(msg), "token") {
				return fmt.Errorf("GitHub token unauthorized. Re-run setup to authenticate or paste a valid Copilot/Models token")
			}
		}
	}
	
	return fmt.Errorf("GitHub provider error: %d %s", statusCode, details)
}