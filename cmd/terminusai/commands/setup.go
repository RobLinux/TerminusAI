package commands

import (
	"terminusai/internal/config"

	"github.com/spf13/cobra"
)

// NewSetupCommand creates the setup command
func NewSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run the interactive setup wizard",
		Long: `The setup command guides you through configuring TerminusAI for first-time use.

It will help you:
- Choose your preferred AI provider (OpenAI, Anthropic, or GitHub Models)
- Configure API credentials securely
- Set default models and preferences
- Test your configuration

Configuration is stored locally in ~/.terminusai/ and includes your API keys,
provider preferences, and command approval policies.`,
		RunE: setupWizard,
		Example: `  terminusai setup
  # Follow the interactive prompts to configure your AI provider`,
	}

	return cmd
}

func setupWizard(cmd *cobra.Command, args []string) error {
	cfg, err := config.SetupWizard(nil)
	if err != nil {
		return err
	}

	green.Println("Setup complete.")
	
	provider := cfg.Provider
	if provider == "" {
		provider = "openai"
	}
	
	model := cfg.Model
	if model != "" {
		white.Printf("Default provider: %s, model: %s\n", provider, model)
	} else {
		white.Printf("Default provider: %s\n", provider)
	}

	return nil
}