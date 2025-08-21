package common

import (
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"AppName", AppName, "terminusai"},
		{"AppVersion", AppVersion, "1.0.0"},
		{"ConfigDirName", ConfigDirName, ".terminusai"},
		{"ConfigFileName", ConfigFileName, "config.json"},
		{"PolicyFileName", PolicyFileName, "policy.json"},
		{"HardcodedGitHubClientID", HardcodedGitHubClientID, "Iv1.b507a08c87ecfe98"},
		{"GitHubOAuthDeviceScopes", GitHubOAuthDeviceScopes, "read:user"},
		{"ProviderOpenAI", ProviderOpenAI, "openai"},
		{"ProviderAnthropic", ProviderAnthropic, "anthropic"},
		{"ProviderGitHub", ProviderGitHub, "github"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.value)
			}
		})
	}
}

func TestTerminusAIConfig(t *testing.T) {
	config := &TerminusAIConfig{
		Provider:            "openai",
		Model:               "gpt-4",
		AlwaysAllow:         true,
		OpenAIAPIKey:        "test-key",
		AnthropicAPIKey:     "test-anthropic-key",
		GitHubToken:         "test-github-token",
		GitHubModelsBaseURL: "https://api.github.com",
		GitHubClientID:      "test-client-id",
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
	if config.OpenAIAPIKey != "test-key" {
		t.Errorf("Expected OpenAIAPIKey to be 'test-key', got %q", config.OpenAIAPIKey)
	}
}

func TestTerminusAIConfigEmpty(t *testing.T) {
	config := &TerminusAIConfig{}

	if config.Provider != "" {
		t.Errorf("Expected empty Provider, got %q", config.Provider)
	}
	if config.AlwaysAllow {
		t.Errorf("Expected AlwaysAllow to be false by default, got %v", config.AlwaysAllow)
	}
}
