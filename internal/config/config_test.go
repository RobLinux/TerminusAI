package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"terminusai/internal/common"
	"testing"
)

func TestTerminusAIConfigAlias(t *testing.T) {
	// Test that TerminusAIConfig is properly aliased
	config := &TerminusAIConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	if config.Provider != "openai" {
		t.Errorf("Expected Provider to be 'openai', got %q", config.Provider)
	}
	if config.Model != "gpt-4" {
		t.Errorf("Expected Model to be 'gpt-4', got %q", config.Model)
	}
}

func TestConfigDir(t *testing.T) {
	dir := configDir()
	if dir == "" {
		t.Errorf("configDir() should not return empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("configDir() should return absolute path, got %q", dir)
	}
	if !contains(dir, common.ConfigDirName) {
		t.Errorf("configDir() should contain %q, got %q", common.ConfigDirName, dir)
	}
}

func TestConfigPath(t *testing.T) {
	path := configPath()
	if path == "" {
		t.Errorf("configPath() should not return empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("configPath() should return absolute path, got %q", path)
	}
	if !contains(path, common.ConfigFileName) {
		t.Errorf("configPath() should contain %q, got %q", common.ConfigFileName, path)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// Test loading config when file doesn't exist
	// First, temporarily change the home directory
	tmpDir, err := ioutil.TempDir("", "terminusai_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	if runtime := os.Getenv("HOME"); runtime != "" {
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", originalHome)
	} else {
		os.Setenv("USERPROFILE", tmpDir)
		defer os.Setenv("USERPROFILE", originalHome)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config == nil {
		t.Fatalf("Expected config to be non-nil")
	}

	// Should return empty config
	if config.Provider != "" {
		t.Errorf("Expected empty Provider, got %q", config.Provider)
	}
}

func TestLoadConfigExisting(t *testing.T) {
	// Create a temporary directory and config file
	tmpDir, err := ioutil.TempDir("", "terminusai_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory
	configDirPath := filepath.Join(tmpDir, common.ConfigDirName)
	err = os.MkdirAll(configDirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write test config
	configFilePath := filepath.Join(configDirPath, common.ConfigFileName)
	configContent := `{
		"provider": "openai",
		"model": "gpt-4",
		"alwaysAllow": true
	}`

	err = ioutil.WriteFile(configFilePath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Mock home directory
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	if runtime := os.Getenv("HOME"); runtime != "" {
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", originalHome)
	} else {
		os.Setenv("USERPROFILE", tmpDir)
		defer os.Setenv("USERPROFILE", originalHome)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.Provider != "openai" {
		t.Errorf("Expected Provider to be 'openai', got %q", config.Provider)
	}
	if config.Model != "gpt-4" {
		t.Errorf("Expected Model to be 'gpt-4', got %q", config.Model)
	}
	if !config.AlwaysAllow {
		t.Errorf("Expected AlwaysAllow to be true, got %v", config.AlwaysAllow)
	}
}

func TestSaveConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "terminusai_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock home directory
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	if runtime := os.Getenv("HOME"); runtime != "" {
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", originalHome)
	} else {
		os.Setenv("USERPROFILE", tmpDir)
		defer os.Setenv("USERPROFILE", originalHome)
	}

	config := &TerminusAIConfig{
		Provider:    "anthropic",
		Model:       "claude-3-sonnet",
		AlwaysAllow: false,
	}

	err = SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	configFilePath := filepath.Join(tmpDir, common.ConfigDirName, common.ConfigFileName)
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created")
	}

	// Load and verify content
	loadedConfig, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Provider != "anthropic" {
		t.Errorf("Expected Provider to be 'anthropic', got %q", loadedConfig.Provider)
	}
	if loadedConfig.Model != "claude-3-sonnet" {
		t.Errorf("Expected Model to be 'claude-3-sonnet', got %q", loadedConfig.Model)
	}
	if loadedConfig.AlwaysAllow {
		t.Errorf("Expected AlwaysAllow to be false, got %v", loadedConfig.AlwaysAllow)
	}
}

func TestKnownModels(t *testing.T) {
	tests := []struct {
		provider string
		expected []string
	}{
		{
			"openai",
			[]string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"},
		},
		{
			"anthropic",
			[]string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"},
		},
		{
			"github",
			[]string{"gpt-4o", "gpt-4o-mini", "Phi-3-medium-128k-instruct", "Phi-3-medium-4k-instruct", "Phi-3-mini-128k-instruct", "Phi-3-mini-4k-instruct", "Phi-3-small-128k-instruct", "Phi-3-small-8k-instruct"},
		},
		{
			"unknown",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := KnownModels(tt.provider)
			if len(models) != len(tt.expected) {
				t.Errorf("Expected %d models for %s, got %d", len(tt.expected), tt.provider, len(models))
				return
			}
			for i, model := range models {
				if model != tt.expected[i] {
					t.Errorf("Expected model %d to be %q, got %q", i, tt.expected[i], model)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > len(substr) && (s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexof(s, substr) >= 0)))
}

func indexof(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
