package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// InteractiveAction represents an action with expandable details
type InteractiveAction struct {
	Title      string
	Summary    string
	Details    []string
	Status     string // "running", "completed", "failed", "skipped"
	StartTime  time.Time
	EndTime    time.Time
	Command    string
	Output     string
	Expandable bool
	Expanded   bool
}

// InteractiveDisplay manages interactive command displays
type InteractiveDisplay struct {
	display *Display
	actions []InteractiveAction
	reader  *bufio.Reader
}

// NewInteractiveDisplay creates a new interactive display manager
func NewInteractiveDisplay(verbose, debug bool) *InteractiveDisplay {
	return &InteractiveDisplay{
		display: NewDisplay(verbose, debug),
		actions: make([]InteractiveAction, 0),
		reader:  bufio.NewReader(os.Stdin),
	}
}

// ShowAction displays an action with interactive capabilities
func (id *InteractiveDisplay) ShowAction(title, summary string, expandable bool) *InteractiveAction {
	action := InteractiveAction{
		Title:      title,
		Summary:    summary,
		StartTime:  time.Now(),
		Status:     "running",
		Expandable: expandable,
		Expanded:   false,
	}

	id.actions = append(id.actions, action)
	actionIndex := len(id.actions) - 1

	// Show the action with bullet point
	Secondary.Printf("● %s\n", title)
	if summary != "" {
		Muted.Printf("  ⎿  %s", summary)
		fmt.Println()
	}

	return &id.actions[actionIndex]
}

// UpdateAction updates an existing action's status and details
func (id *InteractiveDisplay) UpdateAction(action *InteractiveAction, status string, details []string) {
	action.Status = status
	action.Details = details
	action.EndTime = time.Now()

	// Clear the previous line and show updated status
	fmt.Print("\r\033[K")

	switch status {
	case "completed":
		Success.Printf("● %s\n", action.Title)
		elapsed := action.EndTime.Sub(action.StartTime)
		if elapsed > time.Millisecond {
			Muted.Printf("  ⎿  %s [%v]\n", action.Summary, elapsed.Round(time.Millisecond))
		} else {
			Muted.Printf("  ⎿  %s\n", action.Summary)
		}
	case "failed":
		Error.Printf("● %s\n", action.Title)
		Error.Printf("  ⎿  Failed: %s\n", action.Summary)
	case "skipped":
		Warning.Printf("● %s\n", action.Title)
		Warning.Printf("  ⎿  Skipped: %s\n", action.Summary)
	}
}

// ShowListFiles displays a list_files action interactively
func (id *InteractiveDisplay) ShowListFiles(path string, itemCount int) *InteractiveAction {
	title := fmt.Sprintf("List files in %s", path)
	summary := fmt.Sprintf("Found %d items", itemCount)
	return id.ShowAction(title, summary, true)
}

// ShowReadFile displays a read_file action interactively
func (id *InteractiveDisplay) ShowReadFile(path string, bytes int) *InteractiveAction {
	title := fmt.Sprintf("Read file %s", path)
	summary := fmt.Sprintf("Read %d bytes", bytes)
	return id.ShowAction(title, summary, true)
}

// ShowSearchFiles displays a search_files action interactively
func (id *InteractiveDisplay) ShowSearchFiles(pattern, path string, matchCount int) *InteractiveAction {
	title := fmt.Sprintf("Search for '%s'", pattern)
	if path != "" && path != "." {
		title += fmt.Sprintf(" in %s", path)
	}
	summary := fmt.Sprintf("Found %d matches", matchCount)
	return id.ShowAction(title, summary, true)
}

// ShowShellCommand displays a shell command action interactively
func (id *InteractiveDisplay) ShowShellCommand(shell, command, reason string) *InteractiveAction {
	title := reason
	if title == "" || title == "Run command" {
		title = fmt.Sprintf("Execute %s command", shell)
	}

	summary := command
	if len(summary) > 60 {
		summary = summary[:57] + "..."
	}

	action := id.ShowAction(title, summary, true)
	action.Command = command
	return action
}

// ShowAgentThinking displays agent analysis phase
func (id *InteractiveDisplay) ShowAgentThinking(task string) *Spinner {
	Primary.Printf("● Analyzing task: %s\n", task)
	spinner := NewSpinner("  ⎿  Planning approach and identifying requirements...")
	spinner.Start()
	return spinner
}

// PromptForExpansion asks user if they want to see detailed output
func (id *InteractiveDisplay) PromptForExpansion(action *InteractiveAction) bool {
	// Don't prompt automatically - let user decide
	// This reduces interruption and makes flow smoother
	return false
}

// ShowExpandedDetails shows the full details of an action
func (id *InteractiveDisplay) ShowExpandedDetails(action *InteractiveAction) {
	if len(action.Details) == 0 && action.Output == "" {
		Muted.Println("  (No additional details available)")
		return
	}

	Muted.Println("  ┌─ Detailed Output:")

	if action.Command != "" {
		Muted.Printf("  │ Command: %s\n", action.Command)
		Muted.Println("  │")
	}

	if action.Output != "" {
		lines := strings.Split(strings.TrimSpace(action.Output), "\n")
		for i, line := range lines {
			if i < 20 { // Limit to first 20 lines
				Muted.Printf("  │ %s\n", line)
			} else if i == 20 {
				Muted.Printf("  │ ... (%d more lines)\n", len(lines)-20)
				break
			}
		}
	}

	if len(action.Details) > 0 {
		for _, detail := range action.Details {
			if len(detail) > 100 {
				Muted.Printf("  │ %s...\n", detail[:97])
			} else {
				Muted.Printf("  │ %s\n", detail)
			}
		}
	}

	Muted.Println("  └─")
	action.Expanded = true
}

// ShowAgentSummary displays a summary at the end of agent execution
func (id *InteractiveDisplay) ShowAgentSummary() {
	if len(id.actions) == 0 {
		return
	}

	completed := 0
	failed := 0
	skipped := 0

	for _, action := range id.actions {
		switch action.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		case "skipped":
			skipped++
		}
	}

	fmt.Println()
	Primary.Printf("▶ Agent Summary\n")
	Muted.Println("─────────────────")

	if completed > 0 {
		Success.Printf("✓ %d actions completed\n", completed)
	}
	if failed > 0 {
		Error.Printf("✗ %d actions failed\n", failed)
	}
	if skipped > 0 {
		Warning.Printf("⚠ %d actions skipped\n", skipped)
	}

	total := completed + failed + skipped
	if total > 0 {
		fmt.Printf("Total: %d actions\n", total)
	}
}

// CompactOutput controls whether to show minimal output
type CompactOutput struct {
	enabled bool
}

// NewCompactOutput creates a new compact output controller
func NewCompactOutput(enabled bool) *CompactOutput {
	return &CompactOutput{enabled: enabled}
}

// ShouldShowDetails determines if detailed output should be shown
func (c *CompactOutput) ShouldShowDetails(itemCount int) bool {
	if !c.enabled {
		return true
	}

	// Show details for small results, summarize large ones
	return itemCount <= 10
}

// FormatItemCount formats item counts for display
func FormatItemCount(count int, itemType string) string {
	if count == 0 {
		return fmt.Sprintf("No %s found", itemType)
	} else if count == 1 {
		return fmt.Sprintf("1 %s", strings.TrimSuffix(itemType, "s"))
	} else {
		return fmt.Sprintf("%d %s", count, itemType)
	}
}

// TruncateForSummary truncates text for summary display
func TruncateForSummary(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Try to break at word boundary
	if maxLength > 10 {
		truncated := text[:maxLength-3]
		if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLength/2 {
			return truncated[:lastSpace] + "..."
		}
	}

	return text[:maxLength-3] + "..."
}

// ShowProgressiveOutput shows output progressively with user control
func (id *InteractiveDisplay) ShowProgressiveOutput(title string, lines []string, maxInitial int) {
	Secondary.Printf("● %s\n", title)

	if len(lines) == 0 {
		Muted.Println("  ⎿  (No output)")
		return
	}

	if len(lines) <= maxInitial {
		// Show all lines
		for _, line := range lines {
			Muted.Printf("  │ %s\n", line)
		}
		return
	}

	// Show first few lines
	for i := 0; i < maxInitial; i++ {
		Muted.Printf("  │ %s\n", lines[i])
	}

	remaining := len(lines) - maxInitial
	Muted.Printf("  ⎿  (%d more lines - press Enter to show, or any other key to skip)\n", remaining)

	input, _ := id.reader.ReadString('\n')
	if strings.TrimSpace(input) == "" {
		// Show remaining lines
		for i := maxInitial; i < len(lines); i++ {
			Muted.Printf("  │ %s\n", lines[i])
		}
	}
}

// CreateInteractivePrompt creates an interactive prompt with options
func (id *InteractiveDisplay) CreateInteractivePrompt(message string, options []string) (string, error) {
	fmt.Printf("\n%s\n", message)

	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}

	fmt.Print("\nSelect an option (1-" + strconv.Itoa(len(options)) + "): ")

	input, err := id.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		return "", fmt.Errorf("invalid selection")
	}

	return options[choice-1], nil
}
