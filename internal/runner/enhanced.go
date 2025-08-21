package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"terminusai/internal/planner"
	"terminusai/internal/policy"
	"terminusai/internal/ui"
)

// EnhancedRunner provides advanced execution capabilities with cancellation and progress tracking
type EnhancedRunner struct {
	display     *ui.Display
	policyStore *policy.Store
	verbose     bool
	debug       bool
	
	// State management
	mu          sync.RWMutex
	isRunning   bool
	currentStep int
	totalSteps  int
	
	// Cancellation support
	ctx        context.Context
	cancelFunc context.CancelFunc
	
	// Progress tracking
	startTime    time.Time
	stepProgress map[int]StepStatus
}

// StepStatus represents the status of a single step
type StepStatus struct {
	Started   time.Time
	Completed time.Time
	Status    string // "pending", "running", "completed", "failed", "skipped"
	Error     error
}

// NewEnhancedRunner creates a new enhanced runner
func NewEnhancedRunner(policyStore *policy.Store, verbose, debug bool) *EnhancedRunner {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EnhancedRunner{
		display:      ui.NewDisplay(verbose, debug),
		policyStore:  policyStore,
		verbose:      verbose,
		debug:        debug,
		ctx:          ctx,
		cancelFunc:   cancel,
		stepProgress: make(map[int]StepStatus),
	}
}

// SetupCancellation sets up Ctrl+C handling for graceful cancellation
func (r *EnhancedRunner) SetupCancellation() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		r.display.PrintWarning("Cancellation requested...")
		r.Cancel()
	}()
}

// Cancel cancels the current execution
func (r *EnhancedRunner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.isRunning {
		r.cancelFunc()
		r.display.PrintWarning("Execution cancelled")
	}
}

// IsCancelled checks if execution was cancelled
func (r *EnhancedRunner) IsCancelled() bool {
	select {
	case <-r.ctx.Done():
		return true
	default:
		return false
	}
}

// RunPlannedCommandsEnhanced executes a plan with enhanced UI and cancellation support
func (r *EnhancedRunner) RunPlannedCommandsEnhanced(plan *planner.Plan) error {
	r.mu.Lock()
	r.isRunning = true
	r.totalSteps = len(plan.Steps)
	r.startTime = time.Now()
	r.mu.Unlock()
	
	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
	}()
	
	// Setup cancellation handling
	r.SetupCancellation()
	
	// Display execution start
	r.display.PrintExecutionStart()
	
	// Create progress bar
	progressBar := ui.NewProgressBar(len(plan.Steps), "Executing plan")
	
	successfulSteps := 0
	
	for i, step := range plan.Steps {
		// Check for cancellation
		if r.IsCancelled() {
			r.display.PrintWarning("Execution was cancelled")
			return fmt.Errorf("execution cancelled by user")
		}
		
		r.mu.Lock()
		r.currentStep = i + 1
		r.stepProgress[i] = StepStatus{
			Started: time.Now(),
			Status:  "running",
		}
		r.mu.Unlock()
		
		// Display step start
		r.display.PrintStepStart(i+1, len(plan.Steps), step.Description)
		r.display.PrintCommand(step.Shell, step.Command)
		
		// Log verbose information
		r.display.PrintVerbose("Step %d: shell=%s cwd=%s", i+1, step.Shell, step.CWD)
		
		// Get approval from policy store
		decision, err := r.policyStore.Approve(step.Command, step.Description)
		if err != nil {
			r.updateStepStatus(i, "failed", fmt.Errorf("failed to get approval: %w", err))
			r.display.PrintStepError(step.Command, err)
			continue
		}
		
		r.display.PrintVerbose("Decision: %s", decision)
		
		// Handle different decisions
		switch decision {
		case policy.DecisionNever, policy.DecisionSkip:
			r.updateStepStatus(i, "skipped", nil)
			r.display.PrintStepSkipped(step.Command, "user decision")
			progressBar.Increment()
			continue
		}
		
		// Handle session mutations
		if step.SessionMutation {
			r.updateStepStatus(i, "completed", nil)
			r.display.PrintSessionMutation(step.Command)
			progressBar.Increment()
			continue
		}
		
		// Execute the command with real-time status
		commandExecutor := ui.NewCommandExecutor(step.Command)
		commandExecutor.Start()
		
		if err := r.executeStep(step, commandExecutor); err != nil {
			commandExecutor.Complete(false)
			r.updateStepStatus(i, "failed", err)
			r.display.PrintStepError(step.Command, err)
			
			// Ask user if they want to continue on error
			choice, promptErr := r.display.PromptForApproval(
				"Continue execution despite error?", 
				"The previous step failed")
			if promptErr != nil || (choice != "y" && choice != "yes") {
				return fmt.Errorf("execution stopped due to error: %w", err)
			}
		} else {
			commandExecutor.Complete(true)
			r.updateStepStatus(i, "completed", nil)
			r.display.PrintStepSuccess(step.Command)
			successfulSteps++
		}
		
		progressBar.Increment()
	}
	
	progressBar.Complete()
	
	// Display completion summary
	r.display.PrintCompletion(successfulSteps, len(plan.Steps))
	
	// Display execution statistics
	r.displayExecutionStats()
	
	return nil
}

// executeStep executes a single step with cancellation support and real-time feedback
func (r *EnhancedRunner) executeStep(step planner.PlanStep, executor *ui.CommandExecutor) error {
	// Determine shell and arguments
	shell, args := r.getShellCommand(step)
	
	// Create command with context for cancellation
	cmd := exec.CommandContext(r.ctx, shell, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if step.CWD != "" {
		cmd.Dir = step.CWD
	}
	
	r.display.PrintDebug("Executing: %s %v", shell, args)
	
	// Update executor status
	executor.UpdateProgress("starting command")
	
	// Execute command
	if err := cmd.Run(); err != nil {
		executor.UpdateProgress("command failed")
		// Check if it was cancelled
		if r.IsCancelled() {
			return fmt.Errorf("command cancelled")
		}
		
		// Handle execution error
		if exitError, ok := err.(*exec.ExitError); ok {
			r.display.PrintVerbose("Command exit code: %d", exitError.ExitCode())
			return fmt.Errorf("command failed with exit code %d", exitError.ExitCode())
		}
		
		return fmt.Errorf("command execution failed: %w", err)
	}
	
	executor.UpdateProgress("command completed")
	r.display.PrintVerbose("Command completed successfully")
	return nil
}

// getShellCommand determines the shell and arguments for a step
func (r *EnhancedRunner) getShellCommand(step planner.PlanStep) (string, []string) {
	switch step.Shell {
	case "powershell":
		return "pwsh", []string{"-c", step.Command}
	case "cmd":
		return "cmd", []string{"/c", step.Command}
	case "bash":
		return "bash", []string{"-c", step.Command}
	default:
		// Fallback to PowerShell on Windows
		return "pwsh", []string{"-c", step.Command}
	}
}

// updateStepStatus updates the status of a step
func (r *EnhancedRunner) updateStepStatus(stepIndex int, status string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if stepStatus, exists := r.stepProgress[stepIndex]; exists {
		stepStatus.Status = status
		stepStatus.Error = err
		stepStatus.Completed = time.Now()
		r.stepProgress[stepIndex] = stepStatus
	}
}

// GetProgress returns the current execution progress
func (r *EnhancedRunner) GetProgress() (current, total int, isRunning bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.currentStep, r.totalSteps, r.isRunning
}

// displayExecutionStats shows execution statistics
func (r *EnhancedRunner) displayExecutionStats() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if !r.verbose && !r.debug {
		return
	}
	
	executionTime := time.Since(r.startTime)
	
	fmt.Println()
	r.display.PrintInfo("Execution completed in %v", executionTime.Round(time.Millisecond))
	
	if r.debug {
		// Show detailed step timing
		r.display.PrintDebug("Step execution details:")
		for i := 0; i < r.totalSteps; i++ {
			if status, exists := r.stepProgress[i]; exists {
				duration := status.Completed.Sub(status.Started)
				r.display.PrintDebug("  Step %d: %s (%v)", i+1, status.Status, duration.Round(time.Millisecond))
			}
		}
	}
}

// GetStepStatus returns the status of all steps
func (r *EnhancedRunner) GetStepStatus() map[int]StepStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	statusCopy := make(map[int]StepStatus)
	for k, v := range r.stepProgress {
		statusCopy[k] = v
	}
	
	return statusCopy
}