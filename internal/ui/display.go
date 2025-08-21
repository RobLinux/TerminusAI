package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Color schemes for professional output
var (
	// Main colors
	Primary   = color.New(color.FgHiBlue, color.Bold)
	Secondary = color.New(color.FgCyan)
	Success   = color.New(color.FgGreen, color.Bold)
	Warning   = color.New(color.FgYellow, color.Bold)
	Error     = color.New(color.FgRed, color.Bold)
	Muted     = color.New(color.FgHiBlack)
	
	// Command display
	CommandLabel = color.New(color.FgHiBlue, color.Bold)
	CommandText  = color.New(color.FgWhite, color.Bold)
	
	// Status indicators
	Processing = color.New(color.FgYellow)
	Complete   = color.New(color.FgGreen)
	Failed     = color.New(color.FgRed)
	
	// Special formatting
	Highlight = color.New(color.FgHiWhite, color.Bold)
	Subtle    = color.New(color.FgHiBlack)
)

// Spinner represents a running operation indicator
type Spinner struct {
	frames   []string
	current  int
	running  bool
	message  string
	mu       sync.Mutex
	stopChan chan struct{}
}

// NewSpinner creates a new spinner with default frames
func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message:  message,
		stopChan: make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			default:
				s.mu.Lock()
				if !s.running {
					s.mu.Unlock()
					return
				}
				
				frame := s.frames[s.current]
				Processing.Printf("\r%s %s", frame, s.message)
				s.current = (s.current + 1) % len(s.frames)
				s.mu.Unlock()
				
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop halts the spinner and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return
	}
	
	s.running = false
	close(s.stopChan)
	
	// Clear the spinner line
	fmt.Print("\r\033[K")
}

// ProgressBar represents a progress indicator
type ProgressBar struct {
	total   int
	current int
	width   int
	label   string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, label string) *ProgressBar {
	return &ProgressBar{
		total: total,
		width: 40,
		label: label,
	}
}

// Update advances the progress bar
func (p *ProgressBar) Update(current int) {
	p.current = current
	p.render()
}

// Increment advances the progress by 1
func (p *ProgressBar) Increment() {
	p.current++
	p.render()
}

// Complete marks the progress as finished
func (p *ProgressBar) Complete() {
	p.current = p.total
	p.render()
	fmt.Println()
}

func (p *ProgressBar) render() {
	percentage := float64(p.current) / float64(p.total)
	filled := int(percentage * float64(p.width))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%)", 
		p.label, 
		Secondary.Sprint(bar), 
		p.current, 
		p.total, 
		percentage*100)
}

// Display handles the main UI operations
type Display struct {
	verbose bool
	debug   bool
}

// NewDisplay creates a new display manager
func NewDisplay(verbose, debug bool) *Display {
	return &Display{
		verbose: verbose,
		debug:   debug,
	}
}

// PrintHeader displays a formatted header
func (d *Display) PrintHeader(title string) {
	border := strings.Repeat("═", len(title)+4)
	Primary.Printf("┌%s┐\n", border)
	Primary.Printf("│  %s  │\n", title)
	Primary.Printf("└%s┘\n", border)
}

// PrintSection displays a section header
func (d *Display) PrintSection(title string) {
	fmt.Println()
	Primary.Printf("▶ %s\n", title)
	Muted.Println(strings.Repeat("─", len(title)+2))
}

// PrintTask displays a task being executed
func (d *Display) PrintTask(description string) {
	fmt.Printf("\n")
	Secondary.Printf("◦ %s\n", description)
}

// PrintCommand displays a command with professional formatting
func (d *Display) PrintCommand(shell, command string) {
	fmt.Printf("\n")
	CommandLabel.Print("└─ ")
	Muted.Printf("[%s] ", shell)
	CommandText.Printf("%s\n", command)
}

// PrintSynthesis shows command synthesis in progress
func (d *Display) PrintSynthesis(task string) *Spinner {
	fmt.Println()
	Primary.Printf("▶ Synthesizing commands for: %s\n", task)
	
	spinner := NewSpinner("Analyzing task and generating execution plan...")
	spinner.Start()
	return spinner
}

// PrintPlan displays the generated plan
func (d *Display) PrintPlan(stepCount int) {
	fmt.Println()
	Success.Printf("✓ Generated execution plan with %d steps\n", stepCount)
}

// PrintExecutionStart indicates execution is beginning
func (d *Display) PrintExecutionStart() {
	fmt.Println()
	Primary.Println("▶ Executing plan...")
}

// PrintStepStart shows a step is starting
func (d *Display) PrintStepStart(step int, total int, description string) {
	fmt.Printf("\n")
	Primary.Printf("[%d/%d] ", step, total)
	fmt.Printf("%s\n", description)
}

// PrintStepSuccess shows successful step completion
func (d *Display) PrintStepSuccess(command string) {
	Success.Printf("✓ %s\n", command)
}

// PrintStepError shows step failure
func (d *Display) PrintStepError(command string, err error) {
	Error.Printf("✗ %s\n", command)
	Error.Printf("  Error: %s\n", err.Error())
}

// PrintStepSkipped shows step was skipped
func (d *Display) PrintStepSkipped(command string, reason string) {
	Warning.Printf("⚠ Skipped: %s\n", command)
	if reason != "" {
		Muted.Printf("  Reason: %s\n", reason)
	}
}

// PrintCompletion shows final completion status
func (d *Display) PrintCompletion(successful, total int) {
	fmt.Println()
	if successful == total {
		Success.Printf("✓ All commands completed successfully (%d/%d)\n", successful, total)
	} else {
		Warning.Printf("⚠ Completed with issues: %d/%d successful\n", successful, total)
	}
}

// PrintDebug shows debug information if debug mode is enabled
func (d *Display) PrintDebug(format string, args ...interface{}) {
	if d.debug {
		Muted.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// PrintVerbose shows verbose information if verbose mode is enabled
func (d *Display) PrintVerbose(format string, args ...interface{}) {
	if d.verbose {
		Subtle.Printf("[VERBOSE] "+format+"\n", args...)
	}
}

// PrintError displays an error message
func (d *Display) PrintError(format string, args ...interface{}) {
	Error.Printf("✗ "+format+"\n", args...)
}

// PrintWarning displays a warning message
func (d *Display) PrintWarning(format string, args ...interface{}) {
	Warning.Printf("⚠ "+format+"\n", args...)
}

// PrintInfo displays an informational message
func (d *Display) PrintInfo(format string, args ...interface{}) {
	Secondary.Printf("ℹ "+format+"\n", args...)
}

// CancellableOperation represents an operation that can be cancelled
type CancellableOperation struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewCancellableOperation creates a new cancellable operation
func NewCancellableOperation() *CancellableOperation {
	ctx, cancel := context.WithCancel(context.Background())
	return &CancellableOperation{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// Context returns the operation's context
func (op *CancellableOperation) Context() context.Context {
	return op.ctx
}

// Cancel cancels the operation
func (op *CancellableOperation) Cancel() {
	op.cancel()
}

// Done returns the done channel
func (op *CancellableOperation) Done() <-chan struct{} {
	return op.done
}

// Complete marks the operation as complete
func (op *CancellableOperation) Complete() {
	close(op.done)
}

// IsCancelled checks if the operation was cancelled
func (op *CancellableOperation) IsCancelled() bool {
	select {
	case <-op.ctx.Done():
		return true
	default:
		return false
	}
}

// PromptForApproval asks for user approval with cancellation support
func (d *Display) PromptForApproval(command, description string) (string, error) {
	fmt.Println()
	Warning.Printf("⚠ Approval required:\n")
	fmt.Printf("  Command: %s\n", command)
	if description != "" {
		fmt.Printf("  Purpose: %s\n", description)
	}
	
	Highlight.Print("\nChoices: ")
	fmt.Print("(y)es, (n)o, (a)lways, ne(v)er, (s)kip, (q)uit: ")
	
	var choice string
	_, err := fmt.Scanln(&choice)
	if err != nil {
		return "", err
	}
	
	return strings.ToLower(strings.TrimSpace(choice)), nil
}

// PrintSessionMutation displays session mutation instructions
func (d *Display) PrintSessionMutation(command string) {
	fmt.Println()
	Warning.Printf("⚠ Session mutation required\n")
	Secondary.Println("Run this command in your current shell:")
	fmt.Println()
	Highlight.Printf("  %s\n", command)
	fmt.Println()
	Muted.Println("(This command needs to modify your current shell session)")
}

// EnableColors ensures colors are enabled
func EnableColors() {
	color.NoColor = false
}

// DisableColors disables colored output
func DisableColors() {
	color.NoColor = true
}

// IsTerminal checks if output is to a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}