package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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


func EnsureEnv(forceSetup bool) (*TerminusAIConfig, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	needsSetup := forceSetup || !(cfg.Provider != "" && (cfg.OpenAIAPIKey != "" || cfg.AnthropicAPIKey != "" || (cfg.Provider == "copilot" && cfg.GitHubToken != "")))

	if needsSetup {
		updated, err := SetupWizard(cfg)
		if err != nil {
			return nil, err
		}
		return updated, nil
	}

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
		Items: []string{"OpenAI", "Anthropic (Claude)", "GitHub Copilot"},
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
	case "GitHub Copilot":
		answers.Provider = "copilot"
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

	case "copilot":
		if err := handleCopilotAuth(&answers, existing); err != nil {
			return nil, err
		}
	}

	// Preferred model (optional) - skip for copilot as it handles model selection during auth
	if answers.Provider != "copilot" {
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
	}

	if err := SaveConfig(&answers); err != nil {
		return nil, err
	}

	return &answers, nil
}

func handleCopilotAuth(answers *TerminusAIConfig, existing *TerminusAIConfig) error {
	token, err := common.CopilotDeviceFlowAuth()
	if err != nil {
		fmt.Printf("GitHub Copilot authentication failed: %v\n", err)
		return err
	}

	answers.GitHubToken = token
	fmt.Println("GitHub Copilot authentication successful!")

	// Fetch available models
	fmt.Println("Fetching available models...")
	models, err := fetchCopilotModels(token)
	if err != nil {
		fmt.Printf("Warning: Could not fetch models: %v\n", err)
		fmt.Println("Using default models...")
		return nil
	}

	if len(models) > 0 {
		// Present model selection
		modelPrompt := promptui.Select{
			Label: "Select preferred Copilot model",
			Items: models,
		}

		_, selectedModel, err := modelPrompt.Run()
		if err != nil {
			fmt.Printf("No model selected, using default\n")
			return nil
		}

		answers.Model = selectedModel
		fmt.Printf("Selected model: %s\n", selectedModel)
	}

	return nil
}


func fetchCopilotModels(token string) ([]string, error) {
	return common.GetCopilotModels(token)
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
	case "copilot":
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

