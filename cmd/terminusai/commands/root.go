package commands

import (
	"fmt"
	"os"
	"strings"

	"terminusai/internal/agent"
	"terminusai/internal/config"
	"terminusai/internal/policy"
	"terminusai/internal/providers"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cyan  = color.New(color.FgCyan)
	green = color.New(color.FgGreen)
	red   = color.New(color.FgRed)
	white = color.New(color.FgWhite)
)

// NewRootCommand creates the root command for TerminusAI
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "terminusai [task...]",
		Short:   "CLI AI agent that understands and executes tasks with your approval",
		Version: "1.0.0",
		Long: `TerminusAI is a powerful CLI tool that uses AI to understand your tasks,
plan the necessary commands, and execute them with your approval.

Simply describe what you want to do and TerminusAI will figure out the commands.
It supports multiple LLM providers (OpenAI, Anthropic, Copilot) and 
includes security features to ensure all commands require explicit user consent.

Examples:
  terminusai "1+1=?"
  terminusai "build this project into an executable"
  terminusai "create a docker image from this directory"`,
		RunE: handleDirectTask,
		Args: cobra.ArbitraryArgs, // Allow any arguments
		Example: `  terminusai "analyze this codebase"
  terminusai "install dependencies and run tests"
  terminusai --provider anthropic "what files are in this directory?"`,
	}

	// Add flags for direct task execution
	rootCmd.Flags().String("provider", "", "LLM provider: openai|anthropic|copilot")
	rootCmd.Flags().String("model", "", "Model ID override")
	rootCmd.Flags().String("working-dir", "", "Working directory for operations")
	rootCmd.Flags().Bool("setup", false, "Run setup wizard before executing")
	rootCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	rootCmd.Flags().Bool("debug", false, "Enable maximum debug logging")

	// Disable completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add utility subcommands
	rootCmd.AddCommand(
		NewSetupCommand(),
		NewModelCommand(),
		NewConfigCommand(),
	)

	return rootCmd
}

// handleDirectTask processes direct task queries to the root command
func handleDirectTask(cmd *cobra.Command, args []string) error {
	// If no args provided, show help
	if len(args) == 0 {
		return cmd.Help()
	}

	task := strings.Join(args, " ")

	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	workingDir, _ := cmd.Flags().GetString("working-dir")
	setup, _ := cmd.Flags().GetBool("setup")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")

	// Get configuration manager
	cm := config.GetConfigManager()

	// Load user configuration
	if err := cm.LoadUserConfig(); err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	// Set runtime options
	cm.SetVerbose(verbose)
	cm.SetDebug(debug)

	if provider != "" {
		cm.SetProviderOverride(provider)
	}

	if model != "" {
		cm.SetModelOverride(model)
	}

	// Handle setup if needed
	if setup || cm.GetUserConfig().Provider == "" {
		userConfig, err := config.SetupWizard(cm.GetUserConfig())
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		cm.SetUserConfig(userConfig)
		if err := cm.SaveUserConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	providerName := cm.GetEffectiveProvider()

	llmProvider, err := providers.NewProviderWithConfig(cm, providerName)
	if err != nil {
		return err
	}

	policyStore, err := policy.Load()
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Show a simple header
	if verbose || debug {
		cyan.Printf("🤖 TerminusAI\n")
		fmt.Printf("Task: %s\n\n", task)
	}

	// Run in agent mode
	err = agent.RunAgentTaskWithWorkingDir(task, llmProvider, policyStore, workingDir, verbose)
	if err != nil {
		return fmt.Errorf("failed to execute task: %w", err)
	}

	return policyStore.Save()
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		red.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
