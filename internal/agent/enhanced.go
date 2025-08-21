package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"terminusai/internal/policy"
	"terminusai/internal/providers"
	"terminusai/internal/ui"
)

const (
	// maxAPIRetries defines the maximum number of retries for API errors
	maxAPIRetries = 3
	// retryDelay is the base delay between retries (exponential backoff)
	retryDelay = 2 * time.Second
)

// isRetryableError checks if an error is worth retrying (API overload, timeout, etc.)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "overloaded") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504")
}

// EnhancedAgent provides an agent with better UI and interactivity
type EnhancedAgent struct {
	provider    providers.LLMProvider
	policyStore *policy.Store
	display     *ui.InteractiveDisplay
	workingDir  string
	verbose     bool
	debug       bool
}

// NewEnhancedAgent creates a new enhanced agent
func NewEnhancedAgent(provider providers.LLMProvider, policyStore *policy.Store, workingDir string, verbose, debug bool) *EnhancedAgent {
	if workingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workingDir = wd
		} else {
			workingDir = "."
		}
	}

	return &EnhancedAgent{
		provider:    provider,
		policyStore: policyStore,
		display:     ui.NewInteractiveDisplay(verbose, debug),
		workingDir:  workingDir,
		verbose:     verbose,
		debug:       debug,
	}
}

// RunTask executes a task with enhanced UI feedback
func (ea *EnhancedAgent) RunTask(task string) error {
	// Show thinking phase
	spinner := ea.display.ShowAgentThinking(task)

	// Initialize conversation
	maxIters := 12
	transcript := []providers.ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Task: %s\nOS: Windows", task)},
	}

	spinner.Stop()

	for i := 0; i < maxIters; i++ {
		// Trim conversation if getting too long
		if len(transcript) > 10 {
			systemMsg := transcript[0]
			userMsg := transcript[1]
			recent := transcript[len(transcript)-6:]
			transcript = append([]providers.ChatMessage{systemMsg, userMsg}, recent...)
		}

		// Get LLM response with retry mechanism
		var raw string
		var err error

		for retryCount := 0; retryCount <= maxAPIRetries; retryCount++ {
			raw, err = ea.provider.Chat(transcript, nil)
			if err == nil {
				break // Success, exit retry loop
			}

			if !isRetryableError(err) {
				// Non-retryable error, exit immediately
				break
			}

			if retryCount < maxAPIRetries {
				// Show discrete retry message only if we're going to retry
				if ea.verbose {
					ui.Warning.Printf("  ⎿  API temporarily unavailable, retrying in %v... (%d/%d)\n",
						retryDelay*time.Duration(retryCount+1), retryCount+1, maxAPIRetries)
				}
				time.Sleep(retryDelay * time.Duration(retryCount+1)) // Exponential backoff
			}
		}

		if err != nil {
			if isRetryableError(err) {
				ui.Error.Printf("● API service temporarily unavailable after %d retries\n", maxAPIRetries)
				ui.Muted.Printf("  ⎿  The LLM provider is experiencing high load. Please try again in a few minutes.\n")
			} else {
				ui.Error.Printf("● Failed to communicate with LLM provider\n")
				ui.Muted.Printf("  ⎿  %v\n", err)
			}
			return fmt.Errorf("failed to get response from provider: %w", err)
		}

		// Parse action
		action, err := parseAgentAction(raw)
		if err != nil {
			errorMsg := fmt.Sprintf("Invalid action format. Error: %s. Raw response: %s. Please return a single JSON action.", err.Error(), truncateString(raw, 200))
			transcript = append(transcript, providers.ChatMessage{Role: "user", Content: errorMsg})
			continue
		}

		// Execute action with enhanced UI
		switch action.Type {
		case "done":
			result := action.Result
			if result == "" {
				result = "Task completed successfully"
			}

			// Just show simple completion message
			fmt.Println()
			ui.Success.Printf("✓ %s\n", result)
			ea.display.ShowAgentSummary()
			return nil

		case "list_files":
			if err := ea.handleListFilesEnhanced(action, &transcript); err != nil {
				return err
			}

		case "read_file":
			if err := ea.handleReadFileEnhanced(action, &transcript); err != nil {
				return err
			}

		case "search_files":
			if err := ea.handleSearchFilesEnhanced(action, &transcript); err != nil {
				return err
			}

		case "shell":
			if err := ea.handleShellEnhanced(action, &transcript); err != nil {
				return err
			}

		default:
			errorMsg := fmt.Sprintf("Unknown action type: %s", action.Type)
			transcript = append(transcript, providers.ChatMessage{Role: "user", Content: errorMsg})
		}
	}

	ea.display.ShowAction("Max iterations reached", "Agent stopped after reaching maximum iterations", false)
	ea.display.ShowAgentSummary()
	return nil
}

// handleListFilesEnhanced handles list_files with enhanced UI
func (ea *EnhancedAgent) handleListFilesEnhanced(action *AgentAction, transcript *[]providers.ChatMessage) error {
	path := action.Path
	if path == "" {
		path = "."
	}
	depth := 0
	if action.Depth != nil {
		depth = *action.Depth
	}

	// Show the action
	actionUI := ea.display.ShowListFiles(path, 0) // Will update count later

	// Execute the listing
	base := filepath.Join(ea.workingDir, path)
	if !filepath.IsAbs(base) {
		abs, err := filepath.Abs(base)
		if err != nil {
			ea.display.UpdateAction(actionUI, "failed", []string{err.Error()})
			return err
		}
		base = abs
	}

	var lines []string
	if err := listDir(base, depth, &lines, base); err != nil {
		lines = append(lines, fmt.Sprintf("(error listing %s: %s)", base, err.Error()))
		ea.display.UpdateAction(actionUI, "failed", []string{err.Error()})
	} else {
		// Update with actual count
		actionUI.Summary = ui.FormatItemCount(len(lines), "items")
		ea.display.UpdateAction(actionUI, "completed", lines)
	}

	// Limit output for LLM
	if len(lines) > 5000 {
		lines = lines[:5000]
	}
	out := strings.Join(lines, "\n")

	// Only show file listing details if specifically requested
	// Don't auto-expand by default to reduce clutter

	// Add to transcript
	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:list_files\n%s", out)},
	)

	return nil
}

// handleReadFileEnhanced handles read_file with enhanced UI
func (ea *EnhancedAgent) handleReadFileEnhanced(action *AgentAction, transcript *[]providers.ChatMessage) error {
	maxBytes := 4000
	if action.MaxBytes != nil {
		maxBytes = *action.MaxBytes
	}

	// Show the action
	actionUI := ea.display.ShowReadFile(action.Path, 0) // Will update with actual bytes

	// Execute the read
	file := filepath.Join(ea.workingDir, action.Path)
	if !filepath.IsAbs(file) {
		abs, err := filepath.Abs(file)
		if err != nil {
			ea.display.UpdateAction(actionUI, "failed", []string{err.Error()})
			return err
		}
		file = abs
	}

	var content string
	data, err := os.ReadFile(file)
	if err != nil {
		// For file not found, just skip quietly instead of showing as failed
		content = fmt.Sprintf("(file not found)")
		actionUI.Summary = "File not found, skipping"
		ea.display.UpdateAction(actionUI, "skipped", []string{"File does not exist"})

		// Add to transcript as skip
		actionJSON, _ := json.Marshal(action)
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:read_file %s not found", action.Path)},
		)
		return nil
	} else {
		content = string(data)
		bytesRead := min(maxBytes, len(content))
		actionUI.Summary = ui.FormatItemCount(bytesRead, "bytes read")
		ea.display.UpdateAction(actionUI, "completed", []string{fmt.Sprintf("File size: %d bytes", len(content))})
	}

	head := truncateString(content, maxBytes)

	// Only show file content if specifically requested (make it less verbose)
	// Don't auto-expand file contents by default

	// Add to transcript
	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:read_file %s\n%s", action.Path, head)},
	)

	return nil
}

// handleShellEnhanced handles shell commands with enhanced UI
func (ea *EnhancedAgent) handleShellEnhanced(action *AgentAction, transcript *[]providers.ChatMessage) error {
	cwd := action.CWD
	if cwd == "" {
		cwd = ea.workingDir
	}

	reason := action.Reason
	if reason == "" {
		reason = "Execute command"
	}

	// Show the command action
	actionUI := ea.display.ShowShellCommand(action.Shell, action.Command, reason)

	// Update status to show we're waiting for approval
	actionUI.Summary = "Waiting for approval..."

	// Get approval
	decision, err := ea.policyStore.Approve(action.Command, reason)
	if err != nil {
		ea.display.UpdateAction(actionUI, "failed", []string{fmt.Sprintf("Approval failed: %s", err.Error())})
		return fmt.Errorf("failed to get approval: %w", err)
	}

	actionJSON, _ := json.Marshal(action)

	if decision == policy.DecisionNever || decision == policy.DecisionSkip {
		ea.display.UpdateAction(actionUI, "skipped", []string{"User declined to execute"})
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: "observation:shell skipped by user"},
		)
		return nil
	}

	// Execute the command
	var shell string
	var args []string

	switch action.Shell {
	case "powershell":
		shell = "pwsh"
		args = []string{"-c", action.Command}
	case "cmd":
		shell = "cmd"
		args = []string{"/c", action.Command}
	case "bash":
		shell = "bash"
		args = []string{"-c", action.Command}
	default:
		shell = "pwsh"
		args = []string{"-c", action.Command}
	}

	cmd := exec.Command(shell, args...)
	if action.CWD != "" {
		cmd.Dir = action.CWD
	} else {
		cmd.Dir = ea.workingDir
	}

	output, err := cmd.CombinedOutput()
	outputStr := truncateString(string(output), 8000)

	if err != nil {
		exitCode := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		// Show failure
		ea.display.UpdateAction(actionUI, "failed", []string{
			fmt.Sprintf("Exit code: %d", exitCode),
			"Command output:",
			outputStr,
		})

		// Show output if available
		if outputStr != "" {
			actionUI.Output = outputStr
			ea.display.ShowExpandedDetails(actionUI)
		}

		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:shell error exit=%d\n%s", exitCode, outputStr)},
		)
	} else {
		// Show success
		summary := "Command completed successfully"
		if outputStr != "" {
			lines := strings.Split(strings.TrimSpace(outputStr), "\n")
			if len(lines) == 1 && len(lines[0]) < 60 {
				summary = lines[0]
			} else {
				summary = fmt.Sprintf("%d lines of output", len(lines))
			}
		}

		actionUI.Summary = ui.TruncateForSummary(summary, 60)
		ea.display.UpdateAction(actionUI, "completed", []string{summary})

		// Don't auto-show command output to reduce clutter
		// User can request details if needed

		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:shell exit=0\n%s", outputStr)},
		)
	}

	return nil
}

// handleSearchFilesEnhanced handles search_files with enhanced UI
func (ea *EnhancedAgent) handleSearchFilesEnhanced(action *AgentAction, transcript *[]providers.ChatMessage) error {
	pattern := action.Pattern
	if pattern == "" {
		return fmt.Errorf("search pattern is required")
	}

	searchPath := action.Path
	if searchPath == "" {
		searchPath = "."
	}

	maxResults := 50
	if action.MaxResults != nil {
		maxResults = *action.MaxResults
	}

	caseSensitive := true
	if action.CaseSensitive != nil {
		caseSensitive = *action.CaseSensitive
	}

	fileTypes := action.FileTypes
	if len(fileTypes) == 0 {
		// Default to common text file types
		fileTypes = []string{"go", "js", "ts", "py", "java", "c", "cpp", "h", "txt", "md", "json", "yaml", "yml", "xml", "html", "css"}
	}

	// Show the action
	actionUI := ea.display.ShowSearchFiles(pattern, searchPath, 0) // Will update count later

	// Perform the search
	results, err := ea.performFileSearch(pattern, searchPath, fileTypes, caseSensitive, maxResults)
	if err != nil {
		ea.display.UpdateAction(actionUI, "failed", []string{err.Error()})

		actionJSON, _ := json.Marshal(action)
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:search_files error\n%s", err.Error())},
		)
		return nil
	}

	// Update with actual count
	actionUI.Summary = ui.FormatItemCount(len(results), "matches")

	// Format results for display
	var displayResults []string
	if len(results) > 0 {
		displayResults = append(displayResults, fmt.Sprintf("Found %d matches for pattern '%s'", len(results), pattern))
		for i, result := range results {
			if i < 10 { // Show first 10 matches in details
				displayResults = append(displayResults, fmt.Sprintf("%s:%d: %s", result.File, result.LineNum, strings.TrimSpace(result.Line)))
			}
		}
		if len(results) > 10 {
			displayResults = append(displayResults, fmt.Sprintf("... and %d more matches", len(results)-10))
		}
	} else {
		displayResults = []string{"No matches found"}
	}

	ea.display.UpdateAction(actionUI, "completed", displayResults)

	// Format results for LLM
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d matches for pattern '%s':\n", len(results), pattern))

	for _, result := range results {
		output.WriteString(fmt.Sprintf("\n%s:%d: %s", result.File, result.LineNum, strings.TrimSpace(result.Line)))
	}

	outputStr := output.String()
	if len(outputStr) > 8000 {
		outputStr = outputStr[:8000] + "\n... (truncated)"
	}

	// Add to transcript
	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:search_files\n%s", outputStr)},
	)

	return nil
}

// performFileSearch performs the actual file search with regex
func (ea *EnhancedAgent) performFileSearch(pattern, searchPath string, fileTypes []string, caseSensitive bool, maxResults int) ([]SearchResult, error) {
	// Compile regex pattern
	var regex *regexp.Regexp
	var err error

	if caseSensitive {
		regex, err = regexp.Compile(pattern)
	} else {
		regex, err = regexp.Compile("(?i)" + pattern)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Create file type map for quick lookup
	typeMap := make(map[string]bool)
	for _, ext := range fileTypes {
		typeMap["."+ext] = true
	}

	var results []SearchResult
	base := filepath.Join(ea.workingDir, searchPath)

	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			// Skip common heavy directories
			dirName := filepath.Base(path)
			if dirName == "node_modules" || dirName == ".git" || dirName == "dist" ||
				dirName == "build" || dirName == "target" || dirName == ".venv" ||
				dirName == "__pycache__" || dirName == "coverage" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := filepath.Ext(path)
		if !typeMap[ext] {
			return nil
		}

		// Skip large files
		if info.Size() > 1024*1024 { // 1MB limit
			return nil
		}

		// Read and search file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if regex.MatchString(line) {
				relPath, _ := filepath.Rel(base, path)
				results = append(results, SearchResult{
					File:    relPath,
					LineNum: lineNum + 1,
					Line:    line,
				})

				if len(results) >= maxResults {
					return fmt.Errorf("max results reached")
				}
			}
		}

		return nil
	})

	if err != nil && err.Error() != "max results reached" {
		return results, err
	}

	return results, nil
}
