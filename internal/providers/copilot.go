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
	"strings"
	"time"

	"terminusai/internal/common"
)

type CopilotProvider struct {
	name          string
	defaultModel  string
	modelOverride string
	token         string
	copilotToken  string
	config        *common.TerminusAIConfig
}

type CopilotRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type CopilotResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type CopilotErrorResponse struct {
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

// Copilot Models API types
type CopilotModelsResponse struct {
	Data   []CopilotModel `json:"data"`
	Object string         `json:"object"`
}

type CopilotModel struct {
	ID           string                   `json:"id"`
	Object       string                   `json:"object"`
	DisplayName  string                   `json:"display_name"`
	Capabilities CopilotModelCapabilities `json:"capabilities"`
}

type CopilotModelCapabilities struct {
	Family   string               `json:"family"`
	Limits   CopilotModelLimits   `json:"limits"`
	Supports CopilotModelSupports `json:"supports"`
	Object   string               `json:"object"`
}

type CopilotModelLimits struct {
	MaxContextWindowTokens int `json:"max_context_window_tokens,omitempty"`
	MaxOutputTokens        int `json:"max_output_tokens,omitempty"`
	MaxPromptTokens        int `json:"max_prompt_tokens,omitempty"`
	MaxInputs              int `json:"max_inputs,omitempty"`
}

type CopilotModelSupports struct {
	ToolCalls         bool `json:"tool_calls,omitempty"`
	ParallelToolCalls bool `json:"parallel_tool_calls,omitempty"`
	Dimensions        bool `json:"dimensions,omitempty"`
}

// Copilot Chat Completions types
type CopilotChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
}

type CopilotChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewCopilotProvider creates a Copilot provider
func NewCopilotProvider(modelOverride string) *CopilotProvider {
	// Legacy provider - token will be obtained via getCopilotAccessToken when needed
	token := ""

	// Set default Copilot model if none specified or if it's just "copilot"
	actualModel := modelOverride
	if modelOverride == "" || modelOverride == "copilot" {
		actualModel = "gpt-4o" // Default Copilot chat model
	}

	return &CopilotProvider{
		name:          "copilot",
		defaultModel:  actualModel,
		modelOverride: modelOverride,
		token:         token,
		config:        nil,
	}
}


func (p *CopilotProvider) Name() string {
	return p.name
}

func (p *CopilotProvider) DefaultModel() string {
	return p.defaultModel
}

func (p *CopilotProvider) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	// Always use Copilot mode
	return p.chatViaCopilot(messages, opts, nil)
}

// ChatWithConfig allows passing configuration for chat requests
func (p *CopilotProvider) ChatWithConfig(messages []ChatMessage, opts *ChatOptions, cfg *common.TerminusAIConfig) (string, error) {
	return p.chatViaCopilot(messages, opts, cfg)
}

// chatViaCopilot handles chat via Copilot chat completions API
func (p *CopilotProvider) chatViaCopilot(messages []ChatMessage, opts *ChatOptions, cfg *common.TerminusAIConfig) (string, error) {
	if err := p.ensureCopilotToken(); err != nil {
		return "", fmt.Errorf("failed to get Copilot token: %w", err)
	}

	// Determine model from config or options
	model := "gpt-4o" // default Copilot model

	if opts != nil && opts.Model != "" {
		model = opts.Model
	} else if p.modelOverride != "" && p.modelOverride != "copilot" {
		model = p.modelOverride
	} else if p.defaultModel != "" && p.defaultModel != "copilot-codex" && p.defaultModel != "copilot" {
		model = p.defaultModel
	}

	// Determine the base URL based on account type
	baseURL := "https://api.githubcopilot.com"

	url := baseURL + "/chat/completions"

	reqBody := CopilotChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false, // Non-streaming for now
	}

	// Handle temperature from options or config
	if opts != nil && opts.Temperature > 0 {
		temp := float64(opts.Temperature)
		reqBody.Temperature = &temp
	}
	// Note: Config-based temperature should be handled by the config-based provider

	// Handle max tokens
	if opts != nil && opts.MaxTokens > 0 {
		reqBody.MaxTokens = &opts.MaxTokens
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on the TypeScript implementation
	req.Header.Set("Authorization", "Bearer "+p.copilotToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CopilotCopilotChat/0.26.7")
	req.Header.Set("Editor-Version", "copilot-chat/0.26.7")
	req.Header.Set("OpenAI-Organization", "github-copilot")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	req.Header.Set("X-Request-Id", fmt.Sprintf("req_%d", time.Now().UnixNano()))

	// Determine if any message is from an agent ("assistant" or "tool") for X-Initiator header
	isAgentCall := false
	for _, msg := range messages {
		if msg.Role == "assistant" || msg.Role == "tool" {
			isAgentCall = true
			break
		}
	}

	if isAgentCall {
		req.Header.Set("X-Initiator", "agent")
	} else {
		req.Header.Set("X-Initiator", "user")
	}

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
		return "", fmt.Errorf("copilot chat API error: %d %s", resp.StatusCode, string(body))
	}

	var chatResp CopilotChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode chat response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Complete implements the CompletionProvider interface for Copilot mode
func (p *CopilotProvider) Complete(prompt string, opts *CompletionOptions) (string, error) {

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

// GetModels fetches available models from Copilot Copilot API
func (p *CopilotProvider) GetModels() (*CopilotModelsResponse, error) {
	if err := p.ensureCopilotToken(); err != nil {
		return nil, fmt.Errorf("failed to get Copilot token: %w", err)
	}

	// Determine the base URL based on account type
	// For individual accounts: https://api.githubcopilot.com
	// For org accounts: https://api.{org}.githubcopilot.com
	baseURL := "https://api.githubcopilot.com"

	url := baseURL + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on the TypeScript implementation
	req.Header.Set("Authorization", "Bearer "+p.copilotToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CopilotCopilotChat/0.26.7")
	req.Header.Set("Editor-Version", "copilot-chat/0.26.7")
	req.Header.Set("OpenAI-Organization", "github-copilot")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	req.Header.Set("X-Request-Id", fmt.Sprintf("req_%d", time.Now().UnixNano()))

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Copilot models API error: %d %s", resp.StatusCode, string(body))
	}

	var modelsResp CopilotModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	return &modelsResp, nil
}

func (p *CopilotProvider) handleError(statusCode int, body []byte) error {
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

func logCopilotRequest(reqBody CopilotRequest, msgCount int, token, url string, debug bool) {
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

	fmt.Printf("[http] Copilot POST %s token=%s body=%s\n", url, tokenPreview, preview)
}

// ensureCopilotToken ensures we have a valid Copilot token
func (p *CopilotProvider) ensureCopilotToken() error {
	if p.copilotToken != "" && p.isCopilotTokenValid() {
		return nil
	}

	return p.refreshCopilotToken()
}

// refreshCopilotToken gets a new Copilot session token
func (p *CopilotProvider) refreshCopilotToken() error {
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

// getCopilotAccessToken gets the Copilot access token from config or file
func (p *CopilotProvider) getCopilotAccessToken() (string, error) {
	// Try config first
	if p.config != nil && p.config.GitHubToken != "" {
		return p.config.GitHubToken, nil
	}

	// Try .copilot_token file
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenFile := filepath.Join(home, ".copilot_token")
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("no access token found. Run 'terminusai setup' and choose Copilot authentication")
	}

	return strings.TrimSpace(string(data)), nil
}

// isCopilotTokenValid checks if the current token is still valid
func (p *CopilotProvider) isCopilotTokenValid() bool {
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
