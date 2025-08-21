package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"terminusai/internal/common"

	"github.com/joho/godotenv"
	"github.com/manifoldco/promptui"
)

// Type alias for convenience
type TerminusAIConfig = common.TerminusAIConfig

func init() {
	// Load .env file if it exists
	godotenv.Load()
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, common.ConfigDirName)
}

func configPath() string {
	return filepath.Join(configDir(), common.ConfigFileName)
}

func LoadConfig() (*TerminusAIConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &TerminusAIConfig{}, nil
		}
		return nil, err
	}

	var cfg TerminusAIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *TerminusAIConfig) error {
	if err := os.MkdirAll(configDir(), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0644)
}

func applyEnvFromConfig(cfg *TerminusAIConfig) {
	if cfg.OpenAIAPIKey != "" {
		os.Setenv("OPENAI_API_KEY", cfg.OpenAIAPIKey)
	}
	if cfg.AnthropicAPIKey != "" {
		os.Setenv("ANTHROPIC_API_KEY", cfg.AnthropicAPIKey)
	}
	if cfg.GitHubToken != "" {
		os.Setenv("GITHUB_TOKEN", cfg.GitHubToken)
	}
	if cfg.GitHubModelsBaseURL != "" {
		os.Setenv("GITHUB_MODELS_BASE_URL", cfg.GitHubModelsBaseURL)
	}
	if cfg.Model != "" {
		os.Setenv("common.EnvDefaultModel", cfg.Model)
	}
	if cfg.Provider != "" {
		os.Setenv("common.EnvDefaultProvider", cfg.Provider)
	}
	if cfg.GitHubClientID != "" {
		os.Setenv("common.EnvGitHubClientID", cfg.GitHubClientID)
	}
}

func EnsureEnv(forceSetup bool) (*TerminusAIConfig, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	needsSetup := forceSetup || !(cfg.Provider != "" && (cfg.OpenAIAPIKey != "" || cfg.AnthropicAPIKey != "" || cfg.GitHubToken != ""))

	if needsSetup {
		updated, err := SetupWizard(cfg)
		if err != nil {
			return nil, err
		}
		applyEnvFromConfig(updated)
		return updated, nil
	}

	applyEnvFromConfig(cfg)
	return cfg, nil
}

func SetupWizard(existing *TerminusAIConfig) (*TerminusAIConfig, error) {
	if existing == nil {
		existing = &TerminusAIConfig{}
	}

	answers := *existing

	// Choose provider
	providerPrompt := promptui.Select{
		Label: "Select default provider",
		Items: []string{"OpenAI", "Anthropic (Claude)", "GitHub (Copilot / Models)"},
	}

	_, providerResult, err := providerPrompt.Run()
	if err != nil {
		return nil, err
	}

	switch providerResult {
	case "OpenAI":
		answers.Provider = "openai"
	case "Anthropic (Claude)":
		answers.Provider = "anthropic"
	case "GitHub (Copilot / Models)":
		answers.Provider = "github"
	}

	// Provider-specific credentials
	switch answers.Provider {
	case "openai":
		prompt := promptui.Prompt{
			Label: "Enter OpenAI API key (sk-...)",
			Mask:  '*',
			Validate: func(input string) error {
				if len(input) < 10 {
					return fmt.Errorf("API key required")
				}
				return nil
			},
			Default: existing.OpenAIAPIKey,
		}
		result, err := prompt.Run()
		if err != nil {
			return nil, err
		}
		answers.OpenAIAPIKey = result

	case "anthropic":
		prompt := promptui.Prompt{
			Label: "Enter Anthropic API key",
			Mask:  '*',
			Validate: func(input string) error {
				if len(input) < 10 {
					return fmt.Errorf("API key required")
				}
				return nil
			},
			Default: existing.AnthropicAPIKey,
		}
		result, err := prompt.Run()
		if err != nil {
			return nil, err
		}
		answers.AnthropicAPIKey = result

	case "github":
		if err := handleGitHubAuth(&answers, existing); err != nil {
			return nil, err
		}
	}

	// Preferred model (optional)
	modelPrompt := promptui.Prompt{
		Label:   fmt.Sprintf("Preferred default model for %s (leave blank to use default)", answers.Provider),
		Default: existing.Model,
	}
	modelResult, err := modelPrompt.Run()
	if err != nil {
		return nil, err
	}
	if modelResult != "" {
		answers.Model = modelResult
	}

	if err := SaveConfig(&answers); err != nil {
		return nil, err
	}

	return &answers, nil
}

func handleGitHubAuth(answers *TerminusAIConfig, existing *TerminusAIConfig) error {
	// Check if GitHub CLI is available
	ghAvailable := true
	if err := exec.Command("gh", "--version").Run(); err != nil {
		ghAvailable = false
	}

	var choices []string
	if ghAvailable {
		choices = append(choices, "GitHub CLI (gh auth login)")
	}
	choices = append(choices,
		"Paste a token (Copilot/Models – not a regular PAT)",
		"Open browser to token settings, then paste token",
		"Sign in with browser (OAuth Device Flow)",
	)

	methodPrompt := promptui.Select{
		Label: "Authenticate with GitHub via",
		Items: choices,
	}

	_, method, err := methodPrompt.Run()
	if err != nil {
		return err
	}

	switch {
	case strings.Contains(method, "GitHub CLI"):
		return handleGHCLI(answers)
	case strings.Contains(method, "browser to token"):
		return handleBrowserToken(answers, existing)
	case strings.Contains(method, "OAuth Device Flow"):
		return handleOAuthDevice(answers, existing)
	default:
		return handlePasteToken(answers, existing)
	}
}

func handleGHCLI(answers *TerminusAIConfig) error {
	// Try to get token from gh auth
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		// Prompt to login
		fmt.Println("Opening GitHub CLI login...")
		loginCmd := exec.Command("gh", "auth", "login")
		loginCmd.Stdin = os.Stdin
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr

		if err := loginCmd.Run(); err != nil {
			fmt.Println("GitHub CLI login failed")
			return handlePasteToken(answers, &TerminusAIConfig{})
		}

		// Retry getting token
		cmd = exec.Command("gh", "auth", "token")
		output, err = cmd.Output()
		if err != nil {
			fmt.Println("Could not get token from GitHub CLI")
			return handlePasteToken(answers, &TerminusAIConfig{})
		}
	}

	answers.GitHubToken = strings.TrimSpace(string(output))
	return nil
}

func handleBrowserToken(answers *TerminusAIConfig, existing *TerminusAIConfig) error {
	fmt.Println("Opening browser to GitHub token settings...")
	if err := openURL("https://github.com/settings/tokens"); err != nil {
		fmt.Println("Failed to open browser. Please visit: https://github.com/settings/tokens")
	}
	fmt.Println("Create or retrieve a token with access to GitHub Models/Copilot, then paste below.")

	return handlePasteToken(answers, existing)
}

func handleOAuthDevice(answers *TerminusAIConfig, existing *TerminusAIConfig) error {
	clientID := existing.GitHubClientID
	if clientID == "" {
		clientID = os.Getenv("common.EnvGitHubClientID")
		if clientID == "" {
			clientID = common.HardcodedGitHubClientID
		}
	}

	token, err := gitHubDeviceFlowAuth(clientID, common.GitHubOAuthDeviceScopes)
	if err != nil {
		fmt.Printf("OAuth Device Flow failed: %v\n", err)
		return handlePasteToken(answers, existing)
	}

	answers.GitHubToken = token
	answers.GitHubClientID = clientID
	return nil
}

func handlePasteToken(answers *TerminusAIConfig, existing *TerminusAIConfig) error {
	fmt.Println("Note: This requires a GitHub Copilot/Models API token provided by your org or obtained via gh CLI.")

	prompt := promptui.Prompt{
		Label:   "Enter GitHub Copilot/Models token (leave blank to choose another provider)",
		Mask:    '*',
		Default: existing.GitHubToken,
	}

	result, err := prompt.Run()
	if err != nil {
		return err
	}

	if result == "" {
		confirmPrompt := promptui.Prompt{
			Label:    "No token provided. Would you like to select a different provider instead? (y/n)",
			Default:  "y",
			Validate: validateYesNo,
		}

		confirm, err := confirmPrompt.Run()
		if err != nil {
			return err
		}

		if strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes" {
			existing.Provider = "openai"
			newCfg, err := SetupWizard(existing)
			if err != nil {
				return err
			}
			*answers = *newCfg
			return nil
		}
	}

	answers.GitHubToken = result

	// Set base URL
	urlPrompt := promptui.Prompt{
		Label:   "GitHub Models base URL (enter to use default)",
		Default: existing.GitHubModelsBaseURL,
	}
	if urlPrompt.Default == "" {
		urlPrompt.Default = "https://models.inference.ai.azure.com"
	}

	urlResult, err := urlPrompt.Run()
	if err != nil {
		return err
	}
	answers.GitHubModelsBaseURL = urlResult

	// Preflight check if we have a token
	if answers.GitHubToken != "" {
		if err := gitHubModelsPreflight(answers.GitHubToken, answers.GitHubModelsBaseURL); err != nil {
			fmt.Printf("GitHub Models preflight failed: %v\n", err)

			confirmPrompt := promptui.Prompt{
				Label:    "GitHub Models seems unavailable. Switch to another provider? (y/n)",
				Default:  "y",
				Validate: validateYesNo,
			}

			confirm, err := confirmPrompt.Run()
			if err != nil {
				return err
			}

			if strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes" {
				existing.Provider = "openai"
				newCfg, err := SetupWizard(existing)
				if err != nil {
					return err
				}
				*answers = *newCfg
				return nil
			}
		}
	}

	return nil
}

func validateYesNo(input string) error {
	input = strings.ToLower(input)
	if input == "y" || input == "yes" || input == "n" || input == "no" {
		return nil
	}
	return fmt.Errorf("please enter y/yes or n/no")
}

func KnownModels(provider string) []string {
	switch provider {
	case "openai":
		return []string{"gpt-4o-mini", "gpt-4o", "o4-mini", "gpt-4.1-mini"}
	case "anthropic":
		return []string{"claude-3-5-sonnet-latest", "claude-3-5-haiku-latest", "claude-3-opus-latest"}
	case "github":
		return []string{"gpt-4o-mini", "gpt-4o"}
	default:
		return []string{}
	}
}

func SetPreferredModel(provider, model string) (*TerminusAIConfig, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	if provider == "" {
		provider = cfg.Provider
		if provider == "" {
			provider = "openai"
		}
	}

	if model == "" {
		models := KnownModels(provider)
		if len(models) > 0 {
			prompt := promptui.Select{
				Label: fmt.Sprintf("Select preferred model for %s", provider),
				Items: models,
			}

			_, model, err = prompt.Run()
			if err != nil {
				return nil, err
			}
		}
	}

	cfg.Provider = provider
	cfg.Model = model

	if err := SaveConfig(cfg); err != nil {
		return nil, err
	}

	applyEnvFromConfig(cfg)
	return cfg, nil
}

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

func gitHubDeviceFlowAuth(clientID, scopeCSV string) (string, error) {
	scopes := strings.Join(strings.Fields(strings.ReplaceAll(scopeCSV, ",", " ")), " ")

	// Create insecure HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Start device flow
	params := url.Values{}
	params.Set("client_id", clientID)
	if scopes != "" {
		params.Set("scope", scopes)
	}

	req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("device/code failed: %d", resp.StatusCode)
	}

	var startResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&startResp); err != nil {
		return "", err
	}

	if startResp.DeviceCode == "" || startResp.UserCode == "" || startResp.VerificationURI == "" {
		return "", fmt.Errorf("invalid device flow response")
	}

	// Copy user code to clipboard for convenience
	if err := common.CopyToClipboard(startResp.UserCode); err == nil {
		fmt.Printf("\nTo authenticate, open %s and enter code: %s (copied to clipboard)\n", startResp.VerificationURI, startResp.UserCode)
	} else {
		fmt.Printf("\nTo authenticate, open %s and enter code: %s\n", startResp.VerificationURI, startResp.UserCode)
	}

	if startResp.VerificationURIComplete != "" {
		fmt.Printf("Quick link: %s\n", startResp.VerificationURIComplete)
		openURL(startResp.VerificationURIComplete)
	} else {
		openURL(startResp.VerificationURI)
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
			return tokenResp.AccessToken, nil
		}
	}
}

func gitHubModelsPreflight(token, baseURL string) error {
	if baseURL == "" {
		baseURL = os.Getenv("GITHUB_MODELS_BASE_URL")
		if baseURL == "" {
			baseURL = "https://models.inference.ai.azure.com"
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := strings.TrimSuffix(baseURL, "/") + "/v1/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)

		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
			Message string `json:"message"`
		}

		if json.Unmarshal(body, &errResp) == nil {
			if errResp.Error.Message != "" {
				msg = errResp.Error.Message
			} else if errResp.Message != "" {
				msg = errResp.Message
			}
		}

		return fmt.Errorf("%d %s", resp.StatusCode, msg)
	}

	return nil
}
