package commands

import (
	"fmt"
	"strings"

	"terminusai/internal/agent"
	"terminusai/internal/config"
	"terminusai/internal/planner"
	"terminusai/internal/policy"
	"terminusai/internal/providers"
	"terminusai/internal/runner"
	"terminusai/internal/ui"

	"github.com/spf13/cobra"
)

// NewAgentCommand creates the agent command
func NewAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent <task...>",
		Short: "Iterative agent mode: the AI will inspect files and run approved commands step-by-step",
		Long: `The agent command runs TerminusAI in interactive mode where the AI can:
- Inspect files and directories in your project
- Analyze code and configurations
- Run commands step-by-step with your approval
- Adapt its approach based on what it discovers

This mode is ideal for complex tasks that require understanding your codebase structure
or when you want the AI to explore and learn about your project before taking action.

Example:
  terminusai agent "analyze this codebase and suggest improvements"
  terminusai agent "debug the failing tests and fix them"`,
		Args: cobra.MinimumNArgs(1),
		RunE: agentTask,
		Example: `  terminusai agent "build this project into an executable"
  terminusai agent --plan-first "set up CI/CD pipeline"
  terminusai agent --verbose "refactor the authentication system"
  terminusai agent -y "install dependencies and build project"`,
	}

	cmd.Flags().String("provider", "", "LLM provider: openai|anthropic|copilot")
	cmd.Flags().String("model", "", "Model ID override")
	cmd.Flags().String("working-dir", "", "Working directory for agent operations")
	cmd.Flags().Bool("setup", false, "Run setup wizard before executing")
	cmd.Flags().Bool("verbose", false, "Enable verbose logging")
	cmd.Flags().Bool("debug", false, "Enable maximum debug logging")
	cmd.Flags().Bool("plan-first", false, "Generate a plan first, then execute steps with approvals")
	cmd.Flags().Bool("dry-run", false, "With --plan-first, show plan only and exit")
	cmd.Flags().BoolP("always-allow", "y", false, "Automatically approve all commands without prompting")

	return cmd
}

func agentTask(cmd *cobra.Command, args []string) error {
	task := strings.Join(args, " ")

	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	workingDir, _ := cmd.Flags().GetString("working-dir")
	setup, _ := cmd.Flags().GetBool("setup")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")
	planFirst, _ := cmd.Flags().GetBool("plan-first")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	alwaysAllow, _ := cmd.Flags().GetBool("always-allow")

	// Initialize UI
	display := ui.NewDisplay(verbose, debug)
	display.PrintInfo("Task: %s", task)

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

	// Set always-allow mode if requested via flag or config
	userConfig := cm.GetUserConfig()
	if alwaysAllow || userConfig.AlwaysAllow {
		policyStore.SetAlwaysAllow(true)
		if verbose {
			if alwaysAllow {
				display.PrintInfo("Always-allow mode enabled via flag - all commands will be approved automatically")
			} else {
				display.PrintInfo("Always-allow mode enabled via config - all commands will be approved automatically")
			}
		}
	}

	// Optional plan-first flow
	if planFirst {
		display.PrintSection("Planning Phase")

		plan, err := planner.PlanCommands(task, llmProvider)
		if err != nil {
			return fmt.Errorf("failed to plan commands: %w", err)
		}

		if dryRun {
			display.PrintSection("Execution Plan (dry-run)")
			for i, step := range plan.Steps {
				display.PrintTask(fmt.Sprintf("Step %d: %s", i+1, step.Description))
				display.PrintCommand(step.Shell, step.Command)
			}
			return policyStore.Save()
		}

		runnerInstance := runner.NewRunner(policyStore, verbose, debug)
		if err := runnerInstance.RunPlannedCommands(plan); err != nil {
			return fmt.Errorf("failed to run commands: %w", err)
		}

		return policyStore.Save()
	}

	// Use Agent with working directory support
	agentInstance := agent.NewAgent(llmProvider, policyStore, workingDir, verbose, debug)
	if err := agentInstance.RunTask(task); err != nil {
		return fmt.Errorf("agent task failed: %w", err)
	}

	return policyStore.Save()
}
