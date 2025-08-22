package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"terminusai/internal/policy"
	"terminusai/internal/providers"
	"terminusai/internal/ui"
)

// handleListFiles handles list_files
func (a *Agent) handleListFiles(action *AgentAction, transcript *[]providers.ChatMessage) error {
	path := action.Path
	if path == "" {
		path = "."
	}
	depth := 0
	if action.Depth != nil {
		depth = *action.Depth
	}

	// Show the action
	actionUI := a.display.ShowListFiles(path, 0) // Will update count later

	// Execute the listing
	base := filepath.Join(a.workingDir, path)
	if !filepath.IsAbs(base) {
		abs, err := filepath.Abs(base)
		if err != nil {
			a.display.UpdateAction(actionUI, "failed", []string{err.Error()})
			return err
		}
		base = abs
	}

	var lines []string
	if err := listDir(base, depth, &lines, base); err != nil {
		lines = append(lines, fmt.Sprintf("(error listing %s: %s)", base, err.Error()))
		a.display.UpdateAction(actionUI, "failed", []string{err.Error()})
	} else {
		// Update with actual count
		actionUI.Summary = ui.FormatItemCount(len(lines), "items")
		a.display.UpdateAction(actionUI, "completed", lines)
	}

	// Limit output for LLM
	if len(lines) > 500 {
		lines = lines[:500]
	}
	out := strings.Join(lines, "\n")

	// Add to transcript
	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:list_files\n%s", out)},
	)

	return nil
}

// handleReadFile handles read_file
func (a *Agent) handleReadFile(action *AgentAction, transcript *[]providers.ChatMessage) error {
	maxBytes := 4000
	if action.MaxBytes != nil {
		maxBytes = *action.MaxBytes
	}

	// Show the action
	actionUI := a.display.ShowReadFile(action.Path, 0) // Will update with actual bytes

	// Execute the read
	file := filepath.Join(a.workingDir, action.Path)
	if !filepath.IsAbs(file) {
		abs, err := filepath.Abs(file)
		if err != nil {
			a.display.UpdateAction(actionUI, "failed", []string{err.Error()})
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
		a.display.UpdateAction(actionUI, "skipped", []string{"File does not exist"})

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
		a.display.UpdateAction(actionUI, "completed", []string{fmt.Sprintf("File size: %d bytes", len(content))})
	}

	head := truncateString(content, maxBytes)

	// Add to transcript
	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:read_file %s\n%s", action.Path, head)},
	)

	return nil
}

// handleShell handles shell commands
func (a *Agent) handleShell(action *AgentAction, transcript *[]providers.ChatMessage) error {
	cwd := action.CWD
	if cwd == "" {
		cwd = a.workingDir
	}

	reason := action.Reason
	if reason == "" {
		reason = "Execute command"
	}

	// Show the command action
	actionUI := a.display.ShowShellCommand(action.Shell, action.Command, reason)

	// Update status to show we're waiting for approval
	actionUI.Summary = "Waiting for approval..."

	// Get approval
	decision, err := a.policyStore.Approve(action.Command, reason)
	if err != nil {
		a.display.UpdateAction(actionUI, "failed", []string{fmt.Sprintf("Approval failed: %s", err.Error())})
		return fmt.Errorf("failed to get approval: %w", err)
	}

	actionJSON, _ := json.Marshal(action)

	if decision == policy.DecisionNever || decision == policy.DecisionSkip {
		a.display.UpdateAction(actionUI, "skipped", []string{"User declined to execute"})
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
		shell = "powershell.exe"
		args = []string{"-Command", action.Command}
	case "cmd":
		shell = "cmd"
		args = []string{"/c", action.Command}
	case "bash":
		shell = "bash"
		args = []string{"-c", action.Command}
	default:
		shell = "powershell.exe"
		args = []string{"-Command", action.Command}
	}

	cmd := exec.Command(shell, args...)
	if action.CWD != "" {
		cmd.Dir = action.CWD
	} else {
		cmd.Dir = a.workingDir
	}

	output, err := cmd.CombinedOutput()
	outputStr := truncateString(string(output), 8000)

	if err != nil {
		exitCode := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		// Show failure
		a.display.UpdateAction(actionUI, "failed", []string{
			fmt.Sprintf("Exit code: %d", exitCode),
			"Command output:",
			outputStr,
		})

		// Show output if available
		if outputStr != "" {
			actionUI.Output = outputStr
			a.display.ShowExpandedDetails(actionUI)
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
		a.display.UpdateAction(actionUI, "completed", []string{summary})

		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:shell exit=0\n%s", outputStr)},
		)
	}

	return nil
}

// handleSearchFiles handles search_files
func (a *Agent) handleSearchFiles(action *AgentAction, transcript *[]providers.ChatMessage) error {
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

	// Show the action
	actionUI := a.display.ShowSearchFiles(pattern, searchPath, 0) // Will update count later

	// Perform the search
	results, err := a.performFileSearch(pattern, searchPath, fileTypes, caseSensitive, maxResults)
	if err != nil {
		a.display.UpdateAction(actionUI, "failed", []string{err.Error()})

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

	a.display.UpdateAction(actionUI, "completed", displayResults)

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

// handleWriteFile handles write_file
func (a *Agent) handleWriteFile(action *AgentAction, transcript *[]providers.ChatMessage) error {
	filePath := action.Path
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(a.workingDir, action.Path)
	}

	operation := "Write"
	if *action.Append {
		operation = "Append to"
	}

	// Show the action with preview of content
	contentPreview := action.Content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100] + "..."
	}
	contentPreview = strings.ReplaceAll(contentPreview, "\n", " ")

	actionUI := a.display.ShowAction(
		fmt.Sprintf("%s file", operation),
		fmt.Sprintf("File: %s", action.Path),
		true, // needs approval
	)

	reason := action.Reason
	if reason == "" {
		if *action.Append {
			reason = fmt.Sprintf("Append content to file %s", action.Path)
		} else {
			reason = fmt.Sprintf("Write content to file %s", action.Path)
		}
	}

	decision, err := a.policyStore.Approve(fmt.Sprintf("write_file %s", action.Path), reason)
	if err != nil {
		a.display.UpdateAction(actionUI, "failed", []string{fmt.Sprintf("Failed to get approval: %s", err.Error())})
		return fmt.Errorf("failed to get approval: %w", err)
	}

	actionJSON, _ := json.Marshal(action)

	if decision == policy.DecisionNever || decision == policy.DecisionSkip {
		a.display.UpdateAction(actionUI, "skipped", []string{"Skipped by user"})
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: "observation:write_file skipped by user"},
		)
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		errorMsg := fmt.Sprintf("Failed to create parent directory: %s", err.Error())
		a.display.UpdateAction(actionUI, "failed", []string{errorMsg})
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:write_file error\n%s", errorMsg)},
		)
		return nil
	}

	var writeErr error
	if *action.Append {
		// Append to file
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			writeErr = err
		} else {
			defer file.Close()
			_, writeErr = file.WriteString(action.Content)
		}
	} else {
		// Write (overwrite) file
		writeErr = os.WriteFile(filePath, []byte(action.Content), 0644)
	}

	if writeErr != nil {
		errorMsg := writeErr.Error()
		a.display.UpdateAction(actionUI, "failed", []string{errorMsg})
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:write_file error\n%s", errorMsg)},
		)
	} else {
		operation := "written"
		if *action.Append {
			operation = "appended"
		}
		successMsg := fmt.Sprintf("Content %s to %s", operation, action.Path)
		a.display.UpdateAction(actionUI, "completed", []string{successMsg})
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:write_file success\n%s", successMsg)},
		)
	}

	return nil
}

// performFileSearch performs the actual file search with regex
func (a *Agent) performFileSearch(pattern, searchPath string, fileTypes []string, caseSensitive bool, maxResults int) ([]SearchResult, error) {
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
	var typeMap map[string]bool
	searchAllFiles := len(fileTypes) == 0

	if !searchAllFiles {
		typeMap = make(map[string]bool)
		for _, ext := range fileTypes {
			typeMap["."+ext] = true
		}
	}

	var results []SearchResult
	base := filepath.Join(a.workingDir, searchPath)

	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			// Skip common heavy directories
			dirName := filepath.Base(path)
			if dirName == "node_modules" || dirName == ".git" || dirName == ".venv" ||
				dirName == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := filepath.Ext(path)
		if !searchAllFiles && !typeMap[ext] {
			return nil
		}

		// Skip large files
		if info.Size() > maxFileSize {
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