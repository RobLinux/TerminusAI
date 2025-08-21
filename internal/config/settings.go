package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"terminusai/internal/common"
)

// RuntimeSettings represents the current runtime configuration
type RuntimeSettings struct {
	// Logging settings
	Verbose bool `json:"verbose"`
	Debug   bool `json:"debug"`
	
	// LLM settings
	Temperature    *float64 `json:"temperature,omitempty"`
	DefaultModel   string   `json:"defaultModel,omitempty"`
	DefaultProvider string  `json:"defaultProvider,omitempty"`
	
	// Provider overrides for current session
	ProviderOverride string `json:"-"` // Not persisted
	ModelOverride    string `json:"-"` // Not persisted
	
	// Session settings
	DryRun    bool `json:"-"` // Not persisted
	SetupMode bool `json:"-"` // Not persisted
}

// GlobalSettings represents application-wide configuration
type GlobalSettings struct {
	// Application metadata
	Version string `json:"version"`
	
	// Default runtime settings that can be overridden
	Defaults RuntimeSettings `json:"defaults"`
	
	// Provider configurations
	Providers map[string]ProviderConfig `json:"providers"`
	
	// Feature flags
	Features FeatureFlags `json:"features"`
}

// ProviderConfig holds configuration for a specific LLM provider
type ProviderConfig struct {
	Enabled      bool              `json:"enabled"`
	APIKey       string            `json:"apiKey,omitempty"`
	BaseURL      string            `json:"baseUrl,omitempty"`
	DefaultModel string            `json:"defaultModel,omitempty"`
	Models       []string          `json:"availableModels,omitempty"`
	Options      map[string]string `json:"options,omitempty"`
}

// FeatureFlags controls experimental or optional features
type FeatureFlags struct {
	AgentMode        bool `json:"agentMode"`
	PlanFirst        bool `json:"planFirst"`
	InteractiveMode  bool `json:"interactiveMode"`
	VerboseLogging   bool `json:"verboseLogging"`
	DebugMode        bool `json:"debugMode"`
}

// ConfigManager handles all configuration operations
type ConfigManager struct {
	mu             sync.RWMutex
	runtimeSettings *RuntimeSettings
	globalSettings  *GlobalSettings
	userConfig     *TerminusAIConfig
	configDir      string
	settingsPath   string
}

var (
	defaultConfigManager *ConfigManager
	configOnce          sync.Once
)

// GetConfigManager returns the singleton configuration manager
func GetConfigManager() *ConfigManager {
	configOnce.Do(func() {
		defaultConfigManager = NewConfigManager()
	})
	return defaultConfigManager
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("cannot determine home directory: %v", err))
	}
	
	configDir := filepath.Join(home, common.ConfigDirName)
	settingsPath := filepath.Join(configDir, "settings.json")
	
	cm := &ConfigManager{
		configDir:    configDir,
		settingsPath: settingsPath,
		runtimeSettings: &RuntimeSettings{
			Verbose: false,
			Debug:   false,
		},
		globalSettings: getDefaultGlobalSettings(),
	}
	
	// Load existing configuration
	cm.loadSettings()
	
	return cm
}

// getDefaultGlobalSettings returns the default global settings
func getDefaultGlobalSettings() *GlobalSettings {
	return &GlobalSettings{
		Version: common.AppVersion,
		Defaults: RuntimeSettings{
			Verbose:         false,
			Debug:          false,
			DefaultProvider: "openai",
		},
		Providers: map[string]ProviderConfig{
			"openai": {
				Enabled:      true,
				DefaultModel: "gpt-4o-mini",
				Models:       []string{"gpt-4o", "gpt-4o-mini", "o1-mini"},
			},
			"anthropic": {
				Enabled:      true,
				DefaultModel: "claude-opus-4-1-20250805",
				Models:       []string{"claude-opus-4-1-20250805", "claude-3-5-sonnet-latest", "claude-3-5-haiku-latest"},
			},
			"github": {
				Enabled:      true,
				DefaultModel: "gpt-4o-mini",
				Models:       []string{"gpt-4o", "gpt-4o-mini"},
				BaseURL:      "https://models.inference.ai.azure.com",
			},
		},
		Features: FeatureFlags{
			AgentMode:       true,
			PlanFirst:       true,
			InteractiveMode: true,
			VerboseLogging:  true,
			DebugMode:       true,
		},
	}
}

// Runtime Settings Methods

// SetVerbose sets the verbose logging flag
func (cm *ConfigManager) SetVerbose(verbose bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.runtimeSettings.Verbose = verbose
}

// IsVerbose returns the current verbose setting
func (cm *ConfigManager) IsVerbose() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.runtimeSettings.Verbose
}

// SetDebug sets the debug logging flag
func (cm *ConfigManager) SetDebug(debug bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.runtimeSettings.Debug = debug
}

// IsDebug returns the current debug setting
func (cm *ConfigManager) IsDebug() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.runtimeSettings.Debug
}

// SetTemperature sets the LLM temperature
func (cm *ConfigManager) SetTemperature(temp float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.runtimeSettings.Temperature = &temp
}

// GetTemperature returns the current temperature setting
func (cm *ConfigManager) GetTemperature() *float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.runtimeSettings.Temperature
}

// SetProviderOverride sets a temporary provider override
func (cm *ConfigManager) SetProviderOverride(provider string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.runtimeSettings.ProviderOverride = provider
}

// GetEffectiveProvider returns the effective provider (override or default)
func (cm *ConfigManager) GetEffectiveProvider() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if cm.runtimeSettings.ProviderOverride != "" {
		return cm.runtimeSettings.ProviderOverride
	}
	
	if cm.userConfig != nil && cm.userConfig.Provider != "" {
		return cm.userConfig.Provider
	}
	
	return cm.globalSettings.Defaults.DefaultProvider
}

// SetModelOverride sets a temporary model override
func (cm *ConfigManager) SetModelOverride(model string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.runtimeSettings.ModelOverride = model
}

// GetEffectiveModel returns the effective model (override or default)
func (cm *ConfigManager) GetEffectiveModel() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if cm.runtimeSettings.ModelOverride != "" {
		return cm.runtimeSettings.ModelOverride
	}
	
	if cm.userConfig != nil && cm.userConfig.Model != "" {
		return cm.userConfig.Model
	}
	
	provider := cm.GetEffectiveProvider()
	if providerConfig, exists := cm.globalSettings.Providers[provider]; exists {
		return providerConfig.DefaultModel
	}
	
	return ""
}

// Provider Configuration Methods

// GetProviderConfig returns the configuration for a specific provider
func (cm *ConfigManager) GetProviderConfig(provider string) (ProviderConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	config, exists := cm.globalSettings.Providers[provider]
	
	// Merge with user config if available
	if cm.userConfig != nil {
		switch provider {
		case "openai":
			if cm.userConfig.OpenAIAPIKey != "" {
				config.APIKey = cm.userConfig.OpenAIAPIKey
			}
		case "anthropic":
			if cm.userConfig.AnthropicAPIKey != "" {
				config.APIKey = cm.userConfig.AnthropicAPIKey
			}
		case "github":
			if cm.userConfig.GitHubToken != "" {
				config.APIKey = cm.userConfig.GitHubToken
			}
			if cm.userConfig.GitHubModelsBaseURL != "" {
				config.BaseURL = cm.userConfig.GitHubModelsBaseURL
			}
		}
	}
	
	return config, exists
}

// GetAvailableProviders returns a list of enabled providers
func (cm *ConfigManager) GetAvailableProviders() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var providers []string
	for name, config := range cm.globalSettings.Providers {
		if config.Enabled {
			providers = append(providers, name)
		}
	}
	return providers
}

// Feature Flags Methods

// IsFeatureEnabled checks if a specific feature is enabled
func (cm *ConfigManager) IsFeatureEnabled(feature string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	switch feature {
	case "agent":
		return cm.globalSettings.Features.AgentMode
	case "plan-first":
		return cm.globalSettings.Features.PlanFirst
	case "interactive":
		return cm.globalSettings.Features.InteractiveMode
	case "verbose":
		return cm.globalSettings.Features.VerboseLogging
	case "debug":
		return cm.globalSettings.Features.DebugMode
	default:
		return false
	}
}

// Persistence Methods

// LoadUserConfig loads the user configuration
func (cm *ConfigManager) LoadUserConfig() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	userConfig, err := LoadConfig()
	if err != nil {
		return err
	}
	
	cm.userConfig = userConfig
	return nil
}

// SaveUserConfig saves the user configuration
func (cm *ConfigManager) SaveUserConfig() error {
	cm.mu.RLock()
	userConfig := cm.userConfig
	cm.mu.RUnlock()
	
	if userConfig == nil {
		return fmt.Errorf("no user config to save")
	}
	
	return SaveConfig(userConfig)
}

// GetUserConfig returns a copy of the user configuration
func (cm *ConfigManager) GetUserConfig() *TerminusAIConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if cm.userConfig == nil {
		return &TerminusAIConfig{}
	}
	
	// Return a copy to prevent external modification
	config := *cm.userConfig
	return &config
}

// SetUserConfig sets the user configuration
func (cm *ConfigManager) SetUserConfig(config *TerminusAIConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.userConfig = config
}

// loadSettings loads settings from disk
func (cm *ConfigManager) loadSettings() {
	if _, err := os.Stat(cm.settingsPath); os.IsNotExist(err) {
		// Settings file doesn't exist, use defaults
		return
	}
	
	data, err := os.ReadFile(cm.settingsPath)
	if err != nil {
		return // Use defaults on error
	}
	
	var settings GlobalSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return // Use defaults on error
	}
	
	cm.mu.Lock()
	cm.globalSettings = &settings
	cm.mu.Unlock()
}

// SaveSettings saves current settings to disk
func (cm *ConfigManager) SaveSettings() error {
	cm.mu.RLock()
	settings := *cm.globalSettings
	cm.mu.RUnlock()
	
	if err := os.MkdirAll(cm.configDir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(cm.settingsPath, data, 0644)
}

// Reset resets all settings to defaults
func (cm *ConfigManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.runtimeSettings = &RuntimeSettings{
		Verbose: false,
		Debug:   false,
	}
	cm.globalSettings = getDefaultGlobalSettings()
	cm.userConfig = nil
}