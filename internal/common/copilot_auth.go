package common

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CopilotDeviceFlowAuth performs GitHub Device Flow authentication for Copilot
func CopilotDeviceFlowAuth() (string, error) {
	// GitHub Copilot Client ID
	clientID := "Iv1.b507a08c87ecfe98"

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	// Start device flow
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("scope", "read:user")

	req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Editor-Version", "Neovim/0.6.1")
	req.Header.Set("Editor-Plugin-Version", "copilot.vim/1.16.0")
	req.Header.Set("User-Agent", "GithubCopilot/1.155.0")
	req.Header.Set("Accept-Encoding", "identity") // Disable compression

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("device/code failed: %d", resp.StatusCode)
	}

	var startResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&startResp); err != nil {
		return "", err
	}

	if startResp.DeviceCode == "" || startResp.UserCode == "" || startResp.VerificationURI == "" {
		return "", fmt.Errorf("invalid device flow response")
	}

	// Copy user code to clipboard for convenience
	if err := CopyToClipboard(startResp.UserCode); err == nil {
		fmt.Printf("\nTo authenticate GitHub Copilot, open %s and enter code: %s (copied to clipboard)\n",
			startResp.VerificationURI, startResp.UserCode)
	} else {
		fmt.Printf("\nTo authenticate GitHub Copilot, open %s and enter code: %s\n",
			startResp.VerificationURI, startResp.UserCode)
	}

	// Automatically open browser
	fmt.Printf("🌐 Opening browser automatically...\n")
	if err := openURL(startResp.VerificationURI); err != nil {
		fmt.Printf("⚠ Could not open browser automatically: %v\n", err)
		fmt.Printf("Please manually open: %s\n", startResp.VerificationURI)
	}

	// Poll for token
	pollParams := url.Values{}
	pollParams.Set("client_id", clientID)
	pollParams.Set("device_code", startResp.DeviceCode)
	pollParams.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	started := time.Now()
	expiresIn := time.Duration(startResp.ExpiresIn) * time.Second
	interval := time.Duration(startResp.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	for {
		if time.Since(started) > expiresIn {
			return "", fmt.Errorf("device code expired")
		}

		time.Sleep(interval)

		req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(pollParams.Encode()))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Editor-Version", "Neovim/0.6.1")
		req.Header.Set("Editor-Plugin-Version", "copilot.vim/1.16.0")
		req.Header.Set("User-Agent", "GithubCopilot/1.155.0")
		req.Header.Set("Accept-Encoding", "identity") // Disable compression

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		var tokenResp struct {
			AccessToken      string `json:"access_token"`
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if tokenResp.Error != "" {
			switch tokenResp.Error {
			case "authorization_pending":
				continue
			case "slow_down":
				interval += 5 * time.Second
				continue
			case "expired_token", "access_denied":
				return "", fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDescription)
			default:
				continue
			}
		}

		if tokenResp.AccessToken != "" {
			// Save token to file
			if err := saveCopilotToken(tokenResp.AccessToken); err != nil {
				fmt.Printf("Warning: failed to save token to file: %v\n", err)
			}
			fmt.Println("GitHub Copilot authentication successful!")
			return tokenResp.AccessToken, nil
		}
	}
}

// openURL opens the specified URL in the default browser
func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", "", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // assume linux
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// CopilotModel represents a Copilot model
type CopilotModel struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	DisplayName string `json:"display_name"`
}

// CopilotModelsResponse represents the response from the Copilot models API
type CopilotModelsResponse struct {
	Data   []CopilotModel `json:"data"`
	Object string         `json:"object"`
}

// GetCopilotModels fetches available models from GitHub Copilot API
func GetCopilotModels(token string) ([]string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.githubcopilot.com/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.26.7")
	req.Header.Set("Editor-Version", "copilot-chat/0.26.7")
	req.Header.Set("OpenAI-Organization", "github-copilot")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	req.Header.Set("X-Request-Id", fmt.Sprintf("req_%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fall back to known models if API call fails
		return []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
			"claude-3.5-sonnet",
		}, nil
	}

	var modelsResp CopilotModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		// Fall back to known models if decode fails
		return []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
			"claude-3.5-sonnet",
		}, nil
	}

	// Extract model IDs from the response
	var modelIDs []string
	for _, model := range modelsResp.Data {
		modelIDs = append(modelIDs, model.ID)
	}

	if len(modelIDs) == 0 {
		// Fall back to known models if no models returned
		return []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
			"claude-3.5-sonnet",
		}, nil
	}

	return modelIDs, nil
}

// saveCopilotToken saves the access token to ~/.copilot_token
func saveCopilotToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	tokenFile := filepath.Join(home, ".copilot_token")
	return os.WriteFile(tokenFile, []byte(token), 0600)
}
