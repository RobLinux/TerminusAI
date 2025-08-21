package commands

import (
	"fmt"
	"strconv"
	"strings"

	"terminusai/internal/common"
	"terminusai/internal/config"

	"github.com/spf13/cobra"
)

// NewConfigCommand creates the config command
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage TerminusAI configuration settings",
		Long: `The config command allows you to view and update TerminusAI configuration settings.
You can set default values for provider, model, and behavioral settings like always-allow mode.

Configuration is stored in ~/.terminusai/config.json and will be used as defaults for future commands.`,
		RunE: configShow,
	}

	// Subcommands
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigGetCommand())
	cmd.AddCommand(newConfigListCommand())

	return cmd
}

// configShow displays current configuration
func configShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Current TerminusAI Configuration:")
	fmt.Println("================================")

	if cfg.Provider != "" {
		fmt.Printf("Provider:      %s\n", cfg.Provider)
	} else {
		fmt.Printf("Provider:      (not set)\n")
	}

	if cfg.Model != "" {
		fmt.Printf("Model:         %s\n", cfg.Model)
	} else {
		fmt.Printf("Model:         (not set)\n")
	}

	fmt.Printf("Always Allow:  %t\n", cfg.AlwaysAllow)

	// Show API key status (but not the actual keys)
	if cfg.OpenAIAPIKey != "" {
		fmt.Printf("OpenAI API:    configured\n")
	} else {
		fmt.Printf("OpenAI API:    not configured\n")
	}

	if cfg.AnthropicAPIKey != "" {
		fmt.Printf("Anthropic API: configured\n")
	} else {
		fmt.Printf("Anthropic API: not configured\n")
	}

	if cfg.GitHubToken != "" {
		fmt.Printf("GitHub Token:  configured\n")
	} else {
		fmt.Printf("GitHub Token:  not configured\n")
	}

	return nil
}

// newConfigSetCommand creates the 'config set' subcommand
func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value that will be used as default for future commands.

Available keys:
  provider        Set default LLM provider (openai|anthropic|github|copilot)
  model          Set default model ID
  always-allow   Set always-allow mode (true|false)

Examples:
  terminusai config set provider anthropic
  terminusai config set model claude-3-sonnet-20240229
  terminusai config set always-allow true`,
		Args: cobra.ExactArgs(2),
		RunE: configSet,
	}
	return cmd
}

// newConfigGetCommand creates the 'config get' subcommand
func newConfigGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `Get a specific configuration value.`,
		Args:  cobra.ExactArgs(1),
		RunE:  configGet,
	}
	return cmd
}

// newConfigListCommand creates the 'config list' subcommand
func newConfigListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration keys",
		Long:  `List all available configuration keys and their current values.`,
		RunE:  configList,
	}
	return cmd
}

// configSet sets a configuration value
func configSet(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	value := args[1]

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "provider":
		if value != common.ProviderOpenAI && value != common.ProviderAnthropic && value != common.ProviderGitHub {
			return fmt.Errorf("invalid provider: %s (must be openai, anthropic, or github)", value)
		}
		cfg.Provider = value
	case "model":
		cfg.Model = value
	case "always-allow":
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for always-allow: %s (must be true or false)", value)
		}
		cfg.AlwaysAllow = boolValue
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration updated: %s = %s\n", key, value)
	return nil
}

// configGet gets a configuration value
func configGet(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "provider":
		if cfg.Provider != "" {
			fmt.Println(cfg.Provider)
		}
	case "model":
		if cfg.Model != "" {
			fmt.Println(cfg.Model)
		}
	case "always-allow":
		fmt.Println(cfg.AlwaysAllow)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

// configList lists all configuration keys
func configList(cmd *cobra.Command, args []string) error {
	fmt.Println("Available configuration keys:")
	fmt.Println("  provider        Default LLM provider (openai|anthropic|github|copilot)")
	fmt.Println("  model          Default model ID")
	fmt.Println("  always-allow   Always allow commands without prompting (true|false)")
	return nil
}
