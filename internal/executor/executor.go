package executor

import (
	"os"
	"os/exec"

	"terminusai/internal/common"
)

// CommandExecutor handles the execution of system commands
type CommandExecutor struct {
	verbose bool
	debug   bool
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		verbose: common.IsEnvTrue(common.EnvVerbose),
		debug:   common.IsEnvTrue(common.EnvDebug),
	}
}

// ExecuteCommand executes a command with the given shell
func (e *CommandExecutor) ExecuteCommand(shell, command, workingDir string) error {
	var cmd *exec.Cmd
	
	switch shell {
	case "powershell":
		cmd = exec.Command("pwsh", "-c", command)
	case "cmd":
		cmd = exec.Command("cmd", "/c", command)
	case "bash":
		cmd = exec.Command("bash", "-c", command)
	default:
		cmd = exec.Command("pwsh", "-c", command) // Default to PowerShell
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ExecuteCommandWithOutput executes a command and returns its output
func (e *CommandExecutor) ExecuteCommandWithOutput(shell, command, workingDir string) ([]byte, error) {
	var cmd *exec.Cmd
	
	switch shell {
	case "powershell":
		cmd = exec.Command("pwsh", "-c", command)
	case "cmd":
		cmd = exec.Command("cmd", "/c", command)
	case "bash":
		cmd = exec.Command("bash", "-c", command)
	default:
		cmd = exec.Command("pwsh", "-c", command) // Default to PowerShell
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	return cmd.CombinedOutput()
}