package commands

import (
	"terminusai/internal/config"

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