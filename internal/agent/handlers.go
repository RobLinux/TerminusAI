package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"terminusai/internal/policy"
	"terminusai/internal/providers"

	"github.com/fatih/color"
)

// handleListFiles handles list_files action
func handleListFiles(action *AgentAction, transcript *[]providers.ChatMessage, workingDir string, verbose bool) error {
	path := action.Path
	if path == "" {
		path = "."
	}
	depth := 0
	if action.Depth != nil {
		depth = *action.Depth
	}

	if verbose {
		fmt.Printf("[agent] action: list_files path=%s depth=%d\n", path, depth)
	}

	base := filepath.Join(workingDir, path)
	if !filepath.IsAbs(base) {
		abs, err := filepath.Abs(base)
		if err != nil {
			return err
		}
		base = abs
	}

	var lines []string
	if err := listDir(base, depth, &lines, base); err != nil {
		lines = append(lines, fmt.Sprintf("(error listing %s: %s)", base, err.Error()))
	}

	// Limit output
	if len(lines) > 5000 {
		lines = lines[:5000]
	}
	out := strings.Join(lines, "\n")

	if verbose {
		fmt.Printf("[agent] observation: %d items (showing up to 5000)\n%s\n", len(lines), out)
	}

	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:list_files\n%s", out)},
	)

	return nil
}

// handleReadFile handles read_file action
func handleReadFile(action *AgentAction, transcript *[]providers.ChatMessage, workingDir string, verbose bool) error {
	maxBytes := 4000
	if action.MaxBytes != nil {
		maxBytes = *action.MaxBytes
	}

	if verbose {
		fmt.Printf("[agent] action: read_file path=%s maxBytes=%d\n", action.Path, maxBytes)
	}

	file := filepath.Join(workingDir, action.Path)
	if !filepath.IsAbs(file) {
		abs, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		file = abs
	}

	var content string
	data, err := os.ReadFile(file)
	if err != nil {
		content = fmt.Sprintf("(error reading file: %s)", err.Error())
	} else {
		content = string(data)
	}

	head := truncateString(content, maxBytes)

	if verbose {
		fmt.Printf("[agent] observation: read %d bytes from %s\n", min(maxBytes, len(content)), action.Path)
	}

	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:read_file %s\n%s", action.Path, head)},
	)

	return nil
}

// handleShell handles shell action
func handleShell(action *AgentAction, transcript *[]providers.ChatMessage, policyStore *policy.Store, workingDir string, verbose bool) error {
	cwd := action.CWD
	if cwd == "" {
		cwd = workingDir
	}

	// Show what action is being requested (always show, not just verbose)
	showShellAction(action, cwd)

	if verbose {
		fmt.Printf("[agent] action: shell %s cwd=%s cmd=%q\n", action.Shell, cwd, action.Command)
	}

	reason := action.Reason
	if reason == "" {
		reason = "Execute command"
	}
	decision, err := policyStore.Approve(action.Command, reason)
	if err != nil {
		return fmt.Errorf("failed to get approval: %w", err)
	}

	if verbose {
		fmt.Printf("[agent] decision: %s\n", decision)
	}

	actionJSON, _ := json.Marshal(action)

	if decision == policy.DecisionNever || decision == policy.DecisionSkip {
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: "observation:shell skipped by user"},
		)
		if verbose {
			fmt.Println("[agent] observation: shell skipped")
		}
		return nil
	}

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
		shell = "pwsh" // Default to PowerShell
		args = []string{"-c", action.Command}
	}

	cmd := exec.Command(shell, args...)
	if action.CWD != "" {
		cmd.Dir = action.CWD
	} else {
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()
	outputStr := truncateString(string(output), 8000)

	if err != nil {
		exitCode := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:shell error exit=%d\n%s", exitCode, outputStr)},
		)

		if verbose {
			fmt.Printf("[agent] observation: shell error exit=%d\n", exitCode)
		}
	} else {
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:shell exit=0\n%s", outputStr)},
		)

		if verbose {
			fmt.Println("[agent] observation: shell exit=0")
		}
	}

	return nil
}

// handleSearchFiles searches for patterns in files
func handleSearchFiles(action *AgentAction, transcript *[]providers.ChatMessage, workingDir string, verbose bool) error {
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
	// If no file types specified, search in all files (no filtering)
	// Users can specify fileTypes to limit search scope if needed

	if verbose {
		fmt.Printf("[agent] action: search_files pattern=%q path=%s fileTypes=%v\n", pattern, searchPath, fileTypes)
	}

	// Show what action is being performed
	showSearchAction(action)

	// Perform the search
	results, err := performFileSearch(pattern, searchPath, fileTypes, caseSensitive, maxResults, workingDir)
	if err != nil {
		if verbose {
			fmt.Printf("[agent] observation: search failed: %s\n", err.Error())
		}

		actionJSON, _ := json.Marshal(action)
		*transcript = append(*transcript,
			providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
			providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:search_files error\n%s", err.Error())},
		)
		return nil
	}

	// Format results
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d matches for pattern '%s':\n", len(results), pattern))

	for _, result := range results {
		output.WriteString(fmt.Sprintf("\n%s:%d: %s", result.File, result.LineNum, strings.TrimSpace(result.Line)))
	}

	outputStr := output.String()
	if len(outputStr) > 8000 {
		outputStr = outputStr[:8000] + "\n... (truncated)"
	}

	if verbose {
		fmt.Printf("[agent] observation: found %d matches\n", len(results))
	}

	actionJSON, _ := json.Marshal(action)
	*transcript = append(*transcript,
		providers.ChatMessage{Role: "assistant", Content: string(actionJSON)},
		providers.ChatMessage{Role: "user", Content: fmt.Sprintf("observation:search_files\n%s", outputStr)},
	)

	return nil
}

// showShellAction displays the shell command action before approval
func showShellAction(action *AgentAction, cwd string) {
	cyan := color.New(color.FgCyan)
	white := color.New(color.FgWhite, color.Bold)
	muted := color.New(color.FgHiBlack)

	fmt.Println()

	// Show purpose if available
	if action.Reason != "" && action.Reason != "Run command" {
		cyan.Printf("● %s\n", action.Reason)
	} else {
		cyan.Printf("● Execute %s command\n", action.Shell)
	}

	// Show command details
	white.Printf("  ⎿  %s\n", action.Command)
	if action.CWD != "" && action.CWD != cwd {
		muted.Printf("      in directory: %s\n", action.CWD)
	}
}

// showSearchAction displays the search action before execution
func showSearchAction(action *AgentAction) {
	cyan := color.New(color.FgCyan)
	white := color.New(color.FgWhite, color.Bold)
	muted := color.New(color.FgHiBlack)

	fmt.Println()

	cyan.Printf("● Search files for pattern\n")
	white.Printf("  ⎿  Pattern: %s\n", action.Pattern)

	if action.Path != "" && action.Path != "." {
		muted.Printf("      Path: %s\n", action.Path)
	}

	if len(action.FileTypes) > 0 {
		muted.Printf("      File types: %s\n", strings.Join(action.FileTypes, ", "))
	}
}
