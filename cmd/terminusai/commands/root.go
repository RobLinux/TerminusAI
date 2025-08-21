package commands

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cyan   = color.New(color.FgCyan)
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	white  = color.New(color.FgWhite)
)

// NewRootCommand creates the root command for TerminusAI
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "terminusai",
		Short:   "CLI AI agent that plans and runs console commands with your approval",
		Version: "1.0.0",
		Long: `TerminusAI is a powerful CLI tool that uses AI to understand your tasks,
plan the necessary commands, and execute them with your approval.

It supports multiple LLM providers (OpenAI, Anthropic, GitHub Models) and 
includes security features to ensure all commands require explicit user consent.`,
	}

	// Add all subcommands
	rootCmd.AddCommand(
		NewRunCommand(),
		NewAgentCommand(),
		NewSetupCommand(),
		NewModelCommand(),
	)

	return rootCmd
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCommand()
	
	if err := rootCmd.Execute(); err != nil {
		red.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}