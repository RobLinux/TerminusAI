package commands

import (
	"fmt"

	"terminusai/internal/common"

	"github.com/spf13/cobra"
)

// NewCopilotCommand creates the copilot command
func NewCopilotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copilot",
		Short: "GitHub Copilot authentication and management",
		Long:  `Authenticate with GitHub Copilot and manage Copilot settings.`,
	}

	// Add subcommands
	cmd.AddCommand(NewCopilotAuthCommand())

	return cmd
}

// NewCopilotAuthCommand creates the copilot auth command
func NewCopilotAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with GitHub Copilot",
		Long: `Authenticate with GitHub Copilot using device flow authentication.

This command will:
- Start the GitHub Device Flow authentication process
- Copy the device code to your clipboard automatically
- Open your browser to the GitHub device activation page
- Wait for you to complete authentication
- Save the access token for future use

After successful authentication, you can use the Copilot completion API with:
  terminusai agent "your prompt" --provider copilot
  terminusai agent "write a function to sort arrays" --provider github --model copilot`,
		RunE: copilotAuth,
		Example: `  terminusai copilot auth
  # Follow the browser prompts to authenticate with GitHub Copilot`,
	}

	return cmd
}

func copilotAuth(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting GitHub Copilot authentication...")

	token, err := common.CopilotDeviceFlowAuth()
	if err != nil {
		return fmt.Errorf("Copilot authentication failed: %w", err)
	}

	fmt.Printf("Successfully authenticated with GitHub Copilot!\n")
	fmt.Printf("You can now use the copilot provider with: --provider copilot\n")

	// Optionally store token in environment or config
	_ = token // Token is already saved to file by CopilotDeviceFlowAuth

	return nil
}
