package providers

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type GitHubProvider struct {
	name          string
	defaultModel  string
	modelOverride string
	token         string
	baseURL       string
	copilotMode   bool
	copilotToken  string
}

type GitHubRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type GitHubResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type GitHubErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

// Copilot-specific types
type CopilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type CopilotCompletionRequest struct {
	Prompt      string            `json:"prompt"`
	Suffix      string            `json:"suffix"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature int               `json:"temperature"`
	TopP        int               `json:"top_p"`
	N           int               `json:"n"`
	Stop        []string          `json:"stop"`
	NWO         string            `json:"nwo"`
	Stream      bool              `json:"stream"`
	Extra       map[string]string `json:"extra"`
}

type CopilotCompletionChoice struct {
	Text string `json:"text"`
}

type CopilotCompletionResponse struct {
	Choices []CopilotCompletionChoice `json:"choices"`
}

func NewGitHubProvider(modelOverride string) *GitHubProvider {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}

	baseURL := os.Getenv("GITHUB_MODELS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://models.inference.ai.azure.com"
	}

	defaultModel := os.Getenv("TERMINUS_AI_DEFAULT_MODEL")
	if defaultModel == "" {
		defaultModel = os.Getenv("GITHUB_MODEL")
		if defaultModel == "" {
			defaultModel = "gpt-4o-mini"
		}
	}

	// Determine if we should use Copilot mode
	copilotMode := false
	if modelOverride == "copilot" || modelOverride == "copilot-codex" ||
		strings.Contains(strings.ToLower(modelOverride), "copilot") {
		copilotMode = true
		defaultModel = "copilot-codex"
	}

	// Check for Copilot-specific environment variables
	if os.Getenv("GITHUB_COPILOT_MODE") == "true" {
		copilotMode = true
	}

	if token == "" && !copilotMode {
		fmt.Println("Warning: GITHUB_TOKEN not set; github provider will fail until configured.")
	}

	return &GitHubProvider{
		name:          "github",
		defaultModel:  defaultModel,
		modelOverride: modelOverride,
		token:         token,
		baseURL:       baseURL,
		copilotMode:   copilotMode,
	}
}

func (p *GitHubProvider) Name() string {
	return p.name
}

func (p *GitHubProvider) DefaultModel() string {
	return p.defaultModel
}

func (p *GitHubProvider) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	// If in Copilot mode, convert chat to completion
	if p.copilotMode {
		return p.chatViaCopilot(messages, opts)
	}

	// Standard GitHub Models chat
	model := p.defaultModel
	if opts != nil && opts.Model != "" {
		model = opts.Model
	} else if p.modelOverride != "" {
		model = p.modelOverride
	}

	url := p.baseURL + "/v1/chat/completions"

	reqBody := GitHubRequest{
		Model:    model,
		Messages: messages,
	}

	// Handle temperature from environment
	if tempStr := os.Getenv("TERMINUS_AI_TEMPERATURE"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			reqBody.Temperature = &temp
		}
	}

	verbose := false // Legacy mode - no logging
	debug := false   // Legacy mode - no logging

	if verbose || debug {
		logGitHubRequest(reqBody, len(messages), p.token, url, debug)
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
	req.Header.Set("Authorization", "Bearer "+p.token)

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

// chatViaCopilot handles chat via Copilot completion API
func (p *GitHubProvider) chatViaCopilot(messages []ChatMessage, opts *ChatOptions) (string, error) {
	// Convert chat messages to a prompt
	var prompt strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt.WriteString("System: " + msg.Content + "\n")
		case "user":
			prompt.WriteString("User: " + msg.Content + "\n")
		case "assistant":
			prompt.WriteString("Assistant: " + msg.Content + "\n")
		}
	}
	prompt.WriteString("Assistant: ")

	// Use completion API
	completionOpts := &CompletionOptions{
		Language:  "text",
		MaxTokens: 1000,
	}

	return p.Complete(prompt.String(), completionOpts)
}

// Complete implements the CompletionProvider interface for Copilot mode
func (p *GitHubProvider) Complete(prompt string, opts *CompletionOptions) (string, error) {
	if !p.copilotMode {
		return "", fmt.Errorf("completion API only available in Copilot mode")
	}

	if err := p.ensureCopilotToken(); err != nil {
		return "", fmt.Errorf("failed to get Copilot token: %w", err)
	}

	// Set default options
	if opts == nil {
		opts = &CompletionOptions{}
	}
	if opts.Language == "" {
		opts.Language = "python"
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 1000
	}

	reqBody := CopilotCompletionRequest{
		Prompt:      prompt,
		Suffix:      opts.Suffix,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		TopP:        1,
		N:           1,
		Stop:        []string{"\n"},
		NWO:         "github/copilot.vim",
		Stream:      true,
		Extra: map[string]string{
			"language": opts.Language,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://copilot-proxy.githubusercontent.com/v1/engines/copilot-codex/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.copilotToken)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Copilot API error: %d %s", resp.StatusCode, string(body))
	}

	// Parse streaming response
	result := strings.Builder{}
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: {") {
			// Parse the completion from the line as JSON
			jsonData := line[6:] // Remove "data: " prefix
			var completion CopilotCompletionResponse
			if err := json.Unmarshal([]byte(jsonData), &completion); err == nil {
				if len(completion.Choices) > 0 && completion.Choices[0].Text != "" {
					result.WriteString(completion.Choices[0].Text)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read streaming response: %w", err)
	}

	return result.String(), nil
}

func (p *GitHubProvider) handleError(statusCode int, body []byte) error {
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

func logGitHubRequest(reqBody GitHubRequest, msgCount int, token, url string, debug bool) {
	var preview string
	if debug {
		previewJSON, _ := json.Marshal(reqBody)
		preview = string(previewJSON)
	} else {
		preview = fmt.Sprintf(`{"model":"%s","messages":"[%d msg]"}`, reqBody.Model, msgCount)
	}

	var tokenPreview string
	if token != "" && len(token) > 6 {
		tokenPreview = token[:6] + "…"
	} else {
		tokenPreview = "unset"
	}

	fmt.Printf("[http] GitHub POST %s token=%s body=%s\n", url, tokenPreview, preview)
}

// ensureCopilotToken ensures we have a valid Copilot token
func (p *GitHubProvider) ensureCopilotToken() error {
	if p.copilotToken != "" && p.isCopilotTokenValid() {
		return nil
	}

	return p.refreshCopilotToken()
}

// refreshCopilotToken gets a new Copilot session token
func (p *GitHubProvider) refreshCopilotToken() error {
	accessToken, err := p.getCopilotAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/copilot_internal/v2/token", nil)
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Editor-Version", "Neovim/0.6.1")
	req.Header.Set("Editor-Plugin-Version", "copilot.vim/1.16.0")
	req.Header.Set("User-Agent", "GithubCopilot/1.155.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed: %d %s", resp.StatusCode, string(body))
	}

	var tokenResp CopilotTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	p.copilotToken = tokenResp.Token
	return nil
}

// getCopilotAccessToken gets the GitHub access token from file or environment
func (p *GitHubProvider) getCopilotAccessToken() (string, error) {
	// Try environment variable first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token, nil
	}

	// Try .copilot_token file
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenFile := filepath.Join(home, ".copilot_token")
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("no access token found. Set GITHUB_TOKEN environment variable or run 'terminusai copilot auth'")
	}

	return strings.TrimSpace(string(data)), nil
}

// isCopilotTokenValid checks if the current token is still valid
func (p *GitHubProvider) isCopilotTokenValid() bool {
	if p.copilotToken == "" {
		return false
	}

	// Try to extract expiration from token
	parts := strings.Split(p.copilotToken, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "exp=") {
			// For now, assume token is valid for simplicity
			// In a real implementation, you'd parse the timestamp
			return true
		}
	}

	return false
}
