package commands

import (
	"encoding/json"
	"fmt"
	"terminusai/internal/config"
	"terminusai/internal/providers"

	"github.com/spf13/cobra"
)

// NewModelCommand creates the model command
func NewModelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Set or override the preferred model",
		Long: `The model command allows you to configure which AI model to use by default
for your chosen provider.

Each provider supports different models:
- OpenAI: gpt-4o, gpt-4o-mini, o4-mini
- Anthropic: claude-3-5-sonnet-latest, claude-3-5-haiku-latest  
- GitHub: gpt-4o, gpt-4o-mini (via GitHub Models)

You can also override the model per command using the --model flag.`,
		RunE: modelConfig,
		Example: `  terminusai model
  terminusai model --provider anthropic
  terminusai model --provider openai --model gpt-4o`,
	}

	cmd.Flags().String("provider", "", "Provider to set model for (defaults to current)")
	cmd.Flags().String("model", "", "Model ID to set (if omitted, you will be prompted)")

	// Add subcommands
	cmd.AddCommand(NewModelListCommand())

	return cmd
}

// NewModelListCommand creates the model list subcommand
func NewModelListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available models for GitHub Copilot",
		Long: `List available models for GitHub Copilot.

This command fetches the current list of available models from the GitHub Copilot API.
It requires valid Copilot authentication (run 'terminusai setup' first).`,
		RunE: listCopilotModels,
		Example: `  terminusai model list
  terminusai model list --json`,
	}

	cmd.Flags().Bool("json", false, "Output raw JSON response")

	return cmd
}

func modelConfig(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")

	cfg, err := config.SetPreferredModel(provider, model)
	if err != nil {
		return err
	}

	green.Printf("Preferred model set for %s: %s\n", cfg.Provider, cfg.Model)
	return nil
}

func listCopilotModels(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Create GitHub provider to access Copilot API
	provider := providers.NewGitHubProvider("copilot")

	models, err := provider.GetModels()
	if err != nil {
		return fmt.Errorf("failed to fetch Copilot models: %w", err)
	}

	if jsonOutput {
		// Output raw JSON
		jsonData, err := json.MarshalIndent(models, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Pretty formatted output
		fmt.Printf("Available GitHub Copilot Models:\n\n")
		for _, model := range models.Data {
			fmt.Printf("• %s\n", model.ID)
			if model.DisplayName != "" && model.DisplayName != model.ID {
				fmt.Printf("  Name: %s\n", model.DisplayName)
			}
			if model.Capabilities.Family != "" {
				fmt.Printf("  Family: %s\n", model.Capabilities.Family)
			}
			if model.Capabilities.Limits.MaxContextWindowTokens > 0 {
				fmt.Printf("  Max Context: %d tokens\n", model.Capabilities.Limits.MaxContextWindowTokens)
			}
			fmt.Println()
		}
	}

	return nil
}
