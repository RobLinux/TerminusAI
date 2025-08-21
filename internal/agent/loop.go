package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"terminusai/internal/config"
	"terminusai/internal/policy"
	"terminusai/internal/providers"

	"github.com/fatih/color"
)

type AgentAction struct {
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	Depth    *int   `json:"depth,omitempty"`
	MaxBytes *int   `json:"maxBytes,omitempty"`
	Shell    string `json:"shell,omitempty"`
	Command  string `json:"command,omitempty"`
	CWD      string `json:"cwd,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Result   string `json:"result,omitempty"`
	// Search fields
	Pattern     string   `json:"pattern,omitempty"`
	FileTypes   []string `json:"fileTypes,omitempty"`
	CaseSensitive *bool  `json:"caseSensitive,omitempty"`
	MaxResults  *int     `json:"maxResults,omitempty"`
}

const SystemPrompt = `You are a goal-oriented command-line agent. Your job is to achieve the user's task efficiently with minimal discovery.

Available tools (use EXACTLY one per response):
- list_files { path: string, depth?: 0-3 } -> list directory contents
- read_file { path: string, maxBytes?: number } -> read a text file  
- search_files { pattern: string, path?: string, fileTypes?: ["go","js","py"], caseSensitive?: boolean, maxResults?: number } -> search for text patterns in files using regex
- shell { shell: "powershell"|"bash"|"cmd", command: string, cwd?: string, reason?: string } -> execute a command (requires approval)
- done { result: string } -> finish task with summary

CRITICAL RULES:
1. MINIMIZE discovery - only explore if absolutely necessary for the task
2. FOCUS on the goal - don't get distracted by tangential information
3. MOVE TO ACTION quickly - prefer executing commands over endless exploration

Task-specific guidance:
- Create executable: Use pkg, nexe, or similar tools after building
- Search tasks: Use search_files to find patterns, functions, or specific code
- Simple tasks: Execute directly without discovery

Shell preference: On Windows use "powershell"

Output format: Return ONLY valid JSON matching one tool schema. No explanations.

Examples:
Task: "build into exe" + see package.json -> {"type":"shell","shell":"powershell","command":"npm install -g pkg","reason":"Install pkg to create executable"}
Task: "git init" -> {"type":"shell","shell":"powershell","command":"git init","reason":"Initialize git repository"}`

func RunAgentTask(task string, provider providers.LLMProvider, policyStore *policy.Store, verbose bool) error {
	maxIters := 12
	transcript := []providers.ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Task: %s\nOS: Windows", task)},
	}

	for i := 0; i < maxIters; i++ {
		// Keep conversation focused by trimming old observations after 10 messages
		if len(transcript) > 10 {
			systemMsg := transcript[0]
			userMsg := transcript[1]
			recent := transcript[len(transcript)-6:]
			transcript = append([]providers.ChatMessage{systemMsg, userMsg}, recent...)
		}

		cm := config.GetConfigManager()
		debug := cm.IsDebug()
		if debug {
			fmt.Println("[llm] agent messages ->")
			transcriptJSON, _ := json.MarshalIndent(transcript, "", "  ")
			fmt.Println(string(transcriptJSON))
		}

		raw, err := provider.Chat(transcript, nil)
		if err != nil {
			return fmt.Errorf("failed to get response from provider: %w", err)
		}

		if debug {
			fmt.Println("[llm] agent raw ->")
			fmt.Println(raw)
		}

		action, err := parseAgentAction(raw)
		if err != nil {
			errorMsg := fmt.Sprintf("Invalid action format. Error: %s. Raw response: %s. Please return a single JSON action.", err.Error(), truncateString(raw, 200))
			if verbose {
				fmt.Printf("[agent] parse error: %s\n", errorMsg)
			}
			transcript = append(transcript, providers.ChatMessage{Role: "user", Content: errorMsg})
			continue
		}

		if debug {
			fmt.Println("[llm] agent action ->")
			actionJSON, _ := json.MarshalIndent(action, "", "  ")
			fmt.Println(string(actionJSON))
		}

		switch action.Type {
		case "done":
			result := action.Result
			if result == "" {
				result = "Done."
			}
			fmt.Println(result)
			return nil

		case "list_files":
			if err := handleListFiles(action, &transcript, verbose); err != nil {
				return err
			}

		case "read_file":
			if err := handleReadFile(action, &transcript, verbose); err != nil {
				return err
			}

		case "search_files":
			if err := handleSearchFiles(action, &transcript, verbose); err != nil {
				return err
			}

		case "shell":
			if err := handleShell(action, &transcript, policyStore, verbose); err != nil {
				return err
			}

		default:
			errorMsg := fmt.Sprintf("Unknown action type: %s", action.Type)
			transcript = append(transcript, providers.ChatMessage{Role: "user", Content: errorMsg})
		}
	}

	fmt.Println("Reached max iterations without completion.")
	return nil
}

func handleListFiles(action *AgentAction, transcript *[]providers.ChatMessage, verbose bool) error {
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

	cwd, _ := os.Getwd()
	base := filepath.Join(cwd, path)
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

func handleReadFile(action *AgentAction, transcript *[]providers.ChatMessage, verbose bool) error {
	maxBytes := 4000
	if action.MaxBytes != nil {
		maxBytes = *action.MaxBytes
	}

	if verbose {
		fmt.Printf("[agent] action: read_file path=%s maxBytes=%d\n", action.Path, maxBytes)
	}

	cwd, _ := os.Getwd()
	file := filepath.Join(cwd, action.Path)
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

func handleShell(action *AgentAction, transcript *[]providers.ChatMessage, policyStore *policy.Store, verbose bool) error {
	cwd := action.CWD
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			cwd = "unknown"
		}
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

func listDir(dir string, depth int, lines *[]string, base string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	heavy := map[string]bool{
		"node_modules": true, ".git": true, "dist": true, "build": true, "out": true,
		".next": true, "coverage": true, ".venv": true, "venv": true, "__pycache__": true,
		"target": true, ".turbo": true, ".cache": true,
	}

	for _, entry := range entries {
		p := filepath.Join(dir, entry.Name())
		rel, err := filepath.Rel(base, p)
		if err != nil {
			rel = p
		}
		if rel == "" {
			rel = "."
		}

		if entry.IsDir() {
			*lines = append(*lines, rel+"/")
			if depth > 0 && !heavy[entry.Name()] {
				listDir(p, depth-1, lines, base)
			}
		} else {
			*lines = append(*lines, rel)
		}
	}

	return nil
}

func parseAgentAction(raw string) (*AgentAction, error) {
	candidates := []string{strings.TrimSpace(raw)}

	// Try fenced code block
	fenceRegex := regexp.MustCompile("(?i)```(?:json)?\\s*([\\s\\S]*?)```")
	if fence := fenceRegex.FindStringSubmatch(raw); len(fence) > 1 {
		candidates = append(candidates, strings.TrimSpace(fence[1]))
	}

	// Scan for JSON-like substring
	s := raw
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		for j := len(s) - 1; j > i; j-- {
			if s[j] != '}' {
				continue
			}
			sub := s[i : j+1]
			candidates = append(candidates, sub)
		}
	}

	// Try each candidate
	for _, c := range candidates {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(c), &obj); err != nil {
			continue
		}

		if obj != nil {
			coerced := coerceActionType(obj)
			action := &AgentAction{}

			// Marshal back to JSON and unmarshal to struct for type safety
			coercedJSON, _ := json.Marshal(coerced)
			if err := json.Unmarshal(coercedJSON, action); err != nil {
				continue
			}

			if err := validateAction(action); err != nil {
				continue
			}

			return action, nil
		}
	}

	return nil, fmt.Errorf("could not extract a valid JSON action")
}

func coerceActionType(obj map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range obj {
		result[k] = v
	}

	if typeVal, exists := obj["type"]; exists {
		if typeStr, ok := typeVal.(string); ok {
			t := strings.ToLower(typeStr)
			switch t {
			case "list-files":
				result["type"] = "list_files"
			case "get-file":
				result["type"] = "read_file"
			case "run-command":
				result["type"] = "shell"
				if _, exists := result["shell"]; !exists {
					result["shell"] = "powershell"
				}
			case "result":
				result["type"] = "done"
				if text, exists := obj["text"]; exists {
					result["result"] = text
				} else if resultVal, exists := obj["result"]; exists {
					result["result"] = resultVal
				} else {
					result["result"] = ""
				}
			}
		}
	} else {
		// Heuristics
		if command, exists := obj["command"]; exists && command != nil {
			result["type"] = "shell"
			if _, exists := result["shell"]; !exists {
				result["shell"] = "powershell"
			}
		} else if resultVal, exists := obj["result"]; exists && resultVal != nil {
			result["type"] = "done"
		} else if pathVal, exists := obj["path"]; exists && pathVal != nil {
			if pathStr, ok := pathVal.(string); ok {
				// If it looks like a file path (has extension) assume read_file
				fileExtRegex := regexp.MustCompile(`\.[a-zA-Z0-9]+$`)
				dotRegex := regexp.MustCompile(`[^\\\/]\.[^\\\/]`)
				isFileLike := fileExtRegex.MatchString(pathStr) ||
					(dotRegex.MatchString(pathStr) && pathStr != ".")

				if isFileLike || obj["maxBytes"] != nil {
					result["type"] = "read_file"
				} else {
					result["type"] = "list_files"
					if _, exists := result["depth"]; !exists {
						result["depth"] = 0
					}
				}
			}
		}
	}

	return result
}

func validateAction(action *AgentAction) error {
	switch action.Type {
	case "list_files":
		if action.Path == "" {
			action.Path = "."
		}
		if action.Depth == nil {
			depth := 0
			action.Depth = &depth
		} else if *action.Depth < 0 || *action.Depth > 3 {
			return fmt.Errorf("depth must be between 0 and 3")
		}
	case "read_file":
		if action.Path == "" {
			return fmt.Errorf("path is required for read_file")
		}
		if action.MaxBytes == nil {
			maxBytes := 4000
			action.MaxBytes = &maxBytes
		} else if *action.MaxBytes < 1 || *action.MaxBytes > 200000 {
			return fmt.Errorf("maxBytes must be between 1 and 200000")
		}
	case "shell":
		if action.Command == "" {
			return fmt.Errorf("command is required for shell")
		}
		if action.Shell == "" {
			action.Shell = "powershell"
		}
		if action.Shell != "powershell" && action.Shell != "bash" && action.Shell != "cmd" {
			return fmt.Errorf("shell must be powershell, bash, or cmd")
		}
	case "search_files":
		if action.Pattern == "" {
			return fmt.Errorf("pattern is required for search_files")
		}
		if action.Path == "" {
			action.Path = "."
		}
		if action.MaxResults == nil {
			maxResults := 50
			action.MaxResults = &maxResults
		}
		if action.CaseSensitive == nil {
			caseSensitive := false // Default to case-insensitive for better usability
			action.CaseSensitive = &caseSensitive
		}
	case "done":
		if action.Result == "" {
			action.Result = ""
		}
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// handleSearchFiles searches for patterns in files
func handleSearchFiles(action *AgentAction, transcript *[]providers.ChatMessage, verbose bool) error {
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
	
	if verbose {
		fmt.Printf("[agent] action: search_files pattern=%q path=%s fileTypes=%v\n", pattern, searchPath, fileTypes)
	}
	
	// Show what action is being performed
	showSearchAction(action)
	
	// Perform the search
	results, err := performFileSearch(pattern, searchPath, fileTypes, caseSensitive, maxResults)
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

// SearchResult represents a search match result
type SearchResult struct {
	File    string
	LineNum int
	Line    string
}

// performFileSearch performs the actual file search with regex
func performFileSearch(pattern, searchPath string, fileTypes []string, caseSensitive bool, maxResults int) ([]SearchResult, error) {
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
	cwd, _ := os.Getwd()
	base := filepath.Join(cwd, searchPath)
	
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
