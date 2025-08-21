package commands

import (
	"fmt"
	"strings"

	"terminusai/internal/config"
	"terminusai/internal/planner"
	"terminusai/internal/policy"
	"terminusai/internal/providers"
	"terminusai/internal/runner"
	"terminusai/internal/ui"

	"github.com/spf13/cobra"
)

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <task...>",
		Short: "Ask the agent to perform a task, it will propose commands to run",
		Long: `The run command asks the AI to plan and execute a series of commands to complete your task.
The AI will analyze your request, create a step-by-step plan, and ask for your approval 
before executing each command.

Example:
  terminusai run "create a docker image from this directory"
  terminusai run "install dependencies and start the development server"`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    runTask,
		Example: `  terminusai run "build the project and run tests"
  terminusai run --dry-run "deploy to production"
  terminusai run --provider anthropic "analyze this codebase"`,
	}

	cmd.Flags().String("provider", "", "LLM provider: openai|anthropic|github")
	cmd.Flags().String("model", "", "Model ID override")
	cmd.Flags().Bool("setup", false, "Run setup wizard before executing")
	cmd.Flags().Bool("verbose", false, "Enable verbose logging")
	cmd.Flags().Bool("debug", false, "Enable maximum debug logging")
	cmd.Flags().Bool("dry-run", false, "Only show the plan, do not execute")

	return cmd
}

func runTask(cmd *cobra.Command, args []string) error {
	task := strings.Join(args, " ")
	
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	setup, _ := cmd.Flags().GetBool("setup")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Initialize enhanced UI
	display := ui.NewDisplay(verbose, debug)
	display.PrintHeader("TerminusAI Enhanced")

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

	// Display task
	display.PrintInfo("Task: %s", task)

	// Use enhanced planner with visual feedback
	enhancedPlanner := planner.NewEnhancedPlanner(verbose, debug)
	plan, err := enhancedPlanner.PlanCommandsWithUI(task, llmProvider)
	if err != nil {
		return fmt.Errorf("failed to plan commands: %w", err)
	}

	if dryRun {
		display.PrintWarning("Dry-run mode - commands will not be executed")
		display.PrintSection("Plan (dry-run)")
		for i, step := range plan.Steps {
			display.PrintTask(fmt.Sprintf("Step %d: %s", i+1, step.Description))
			display.PrintCommand(step.Shell, step.Command)
		}
		return nil
	}

	// Use enhanced runner with cancellation and progress tracking
	enhancedRunner := runner.NewEnhancedRunner(policyStore, verbose, debug)
	if err := enhancedRunner.RunPlannedCommandsEnhanced(plan); err != nil {
		return fmt.Errorf("failed to run commands: %w", err)
	}

	return policyStore.Save()
}