package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terminusai/internal/common"
)

type OpenAIProvider struct {
	name         string
	defaultModel string
	modelOverride string
	apiKey       string
	config       *common.TerminusAIConfig
}

type OpenAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewOpenAIProvider(modelOverride string) *OpenAIProvider {
	defaultModel := "gpt-4o-mini"
	
	return &OpenAIProvider{
		name:          "openai",
		defaultModel:  defaultModel,
		modelOverride: modelOverride,
		apiKey:        "", // Legacy provider - should use config-based provider instead
		config:        nil,
	}
}


func (p *OpenAIProvider) Name() string {
	return p.name
}

func (p *OpenAIProvider) DefaultModel() string {
	return p.defaultModel
}

func (p *OpenAIProvider) Chat(messages []ChatMessage, opts *ChatOptions) (string, error) {
	model := p.defaultModel
	if opts != nil && opts.Model != "" {
		model = opts.Model
	} else if p.modelOverride != "" {
		model = p.modelOverride
	}

	reqBody := OpenAIRequest{
		Model:    model,
		Messages: messages,
	}

	// Legacy provider uses default temperature handling from options only

	verbose := false  // Legacy mode - no logging
	debug := false    // Legacy mode - no logging

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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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



func logRequest(reqBody OpenAIRequest, debug bool) {
	var msgPreview []ChatMessage
	if debug {
		msgPreview = reqBody.Messages
	} else {
		for _, msg := range reqBody.Messages {
			msgPreview = append(msgPreview, ChatMessage{
				Role:    msg.Role,
				Content: common.TruncateString(msg.Content, 1000),
			})
		}
	}
	
	preview := OpenAIRequest{
		Model:       reqBody.Model,
		Messages:    msgPreview,
		Temperature: reqBody.Temperature,
	}
	
	previewJSON, _ := json.Marshal(preview)
	fmt.Printf("[http] OpenAI POST /chat/completions model=%s body=%s\n", reqBody.Model, string(previewJSON))
}

func logResponse(text string, debug bool) {
	var out string
	if debug {
		out = text
	} else {
		out = common.TruncateString(text, 2000)
		if len(text) > 2000 {
			out += "…"
		}
	}
	fmt.Printf("[http] OpenAI <- %s\n", out)
}