package providers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type GitHubProvider struct {
	name          string
	defaultModel  string
	modelOverride string
	token         string
	baseURL       string
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
	
	if token == "" {
		fmt.Println("Warning: GITHUB_TOKEN not set; github provider will fail until configured.")
	}
	
	return &GitHubProvider{
		name:          "github",
		defaultModel:  defaultModel,
		modelOverride: modelOverride,
		token:         token,
		baseURL:       baseURL,
	}
}

func (p *GitHubProvider) Name() string {
	return p.name
}

func (p *GitHubProvider) DefaultModel() string {
	return p.defaultModel
}

func (p *GitHubProvider) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
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

	verbose := false  // Legacy mode - no logging
	debug := false    // Legacy mode - no logging

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