package common

// TerminusAIConfig represents the application configuration
type TerminusAIConfig struct {
	Provider             string `json:"provider,omitempty"`
	Model                string `json:"model,omitempty"`
	OpenAIAPIKey         string `json:"openaiApiKey,omitempty"`
	AnthropicAPIKey      string `json:"anthropicApiKey,omitempty"`
	GitHubToken          string `json:"githubToken,omitempty"`
	GitHubModelsBaseURL  string `json:"githubModelsBaseUrl,omitempty"`
	GitHubClientID       string `json:"githubClientId,omitempty"`
}

// Constants for the application
const (
	AppName                   = "terminusai"
	AppVersion               = "1.0.0"
	ConfigDirName            = ".terminusai"
	ConfigFileName           = "config.json"
	PolicyFileName           = "policy.json"
	HardcodedGitHubClientID  = "Ov23li7S3AOiOVyDxiV4"
	GitHubOAuthDeviceScopes  = "read:user"
)

// Supported LLM providers
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGitHub    = "github"
)