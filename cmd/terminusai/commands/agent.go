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
	cmd.Flags().Bool("minimal", false, "Use minimal output (disable enhanced UI)")
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
	minimal, _ := cmd.Flags().GetBool("minimal")
	planFirst, _ := cmd.Flags().GetBool("plan-first")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	alwaysAllow, _ := cmd.Flags().GetBool("always-allow")

	// Initialize enhanced UI
	display := ui.NewDisplay(verbose, debug)
	if !minimal {
		// More minimal header
		display.PrintInfo("Task: %s", task)
	}

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
		if minimal {
			cyan.Printf("Task: %s\n", task)
		} else {
			display.PrintSection("Planning Phase")
		}

		plan, err := planner.PlanCommands(task, llmProvider)
		if err != nil {
			return fmt.Errorf("failed to plan commands: %w", err)
		}

		if dryRun {
			if minimal {
				yellow.Println("\nPlan (dry-run):")
				for _, step := range plan.Steps {
					fmt.Printf("- %s: %s\n", step.Shell, step.Command)
				}
			} else {
				display.PrintSection("Execution Plan (dry-run)")
				for i, step := range plan.Steps {
					display.PrintTask(fmt.Sprintf("Step %d: %s", i+1, step.Description))
					display.PrintCommand(step.Shell, step.Command)
				}
			}
			return policyStore.Save()
		}

		if minimal {
			if err := runner.RunPlannedCommands(plan, policyStore, cm.IsVerbose()); err != nil {
				return fmt.Errorf("failed to run commands: %w", err)
			}
		} else {
			enhancedRunner := runner.NewEnhancedRunner(policyStore, verbose, debug)
			if err := enhancedRunner.RunPlannedCommandsEnhanced(plan); err != nil {
				return fmt.Errorf("failed to run commands: %w", err)
			}
		}
		return policyStore.Save()
	}

	// Always use enhanced functionality with working directory support
	if err := agent.RunAgentTaskWithWorkingDir(task, llmProvider, policyStore, workingDir, cm.IsVerbose()); err != nil {
		return fmt.Errorf("agent task failed: %w", err)
	}

	return policyStore.Save()
}
