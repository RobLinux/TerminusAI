package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"terminusai/internal/planner"
	"terminusai/internal/policy"

	"github.com/fatih/color"
)

var (
	cyan   = color.New(color.FgCyan)
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	white  = color.New(color.FgWhite)
)

func RunPlannedCommands(plan *planner.Plan, policyStore *policy.Store, verbose bool) error {
	for i, step := range plan.Steps {
		if verbose {
			cwd := step.CWD
			if cwd == "" {
				if wd, err := os.Getwd(); err == nil {
					cwd = wd
				} else {
					cwd = "unknown"
				}
			}
			fmt.Printf("[run] step: shell=%s cwd=%s cmd=%q\n", step.Shell, cwd, step.Command)
		}

		decision, err := policyStore.Approve(step.Command, step.Description)
		if err != nil {
			return fmt.Errorf("failed to get approval for step %d: %w", i, err)
		}

		if verbose {
			fmt.Printf("[run] decision: %s\n", decision)
		}

		if decision == policy.DecisionNever || decision == policy.DecisionSkip {
			yellow.Printf("Skipped: %s\n", step.Command)
			continue
		}

		// Session mutating commands cannot affect the parent shell; print guidance and a helper snippet.
		if step.SessionMutation {
			cyan.Println("\nSession mutation required. Run this in your current shell:")
			white.Printf("  %s\n", step.Command)
			// For 'always', we just won't re-prompt next time; no execution here.
			continue
		}

		green.Printf("\n> %s\n", step.Command)

		var shell string
		var args []string

		switch step.Shell {
		case "powershell":
			shell = "powershell.exe"
			args = []string{"-Command", step.Command}
		case "cmd":
			shell = "cmd"
			args = []string{"/c", step.Command}
		case "bash":
			shell = "bash"
			args = []string{"-c", step.Command}
		default:
			return fmt.Errorf("unsupported shell: %s", step.Shell)
		}

		cmd := exec.Command(shell, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if step.CWD != "" {
			cmd.Dir = step.CWD
		}

		if err := cmd.Run(); err != nil {
			red.Printf("Command failed: %s\n", step.Command)
			if exitError, ok := err.(*exec.ExitError); ok {
				red.Printf("Exit code: %d\n", exitError.ExitCode())
				if verbose {
					fmt.Printf("[run] result: exit=%d\n", exitError.ExitCode())
				}
			}
			return fmt.Errorf("command failed: %w", err)
		}

		if verbose {
			fmt.Println("[run] result: exit=0")
		}
	}

	return nil
}

func isEnvTrue(envVar string) bool {
	val := strings.ToLower(os.Getenv(envVar))
	return val == "1" || val == "true"
}
