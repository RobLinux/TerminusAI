package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"terminusai/internal/config"
	"terminusai/internal/policy"
	"terminusai/internal/providers"
)

const SystemPrompt = `You are a goal-oriented command-line agent. Your job is to achieve the user's task efficiently with minimal discovery.

Available tools (use EXACTLY one per response):
- list_files { path: string, depth?: 0-3 } -> list directory contents
- read_file { path: string, maxBytes?: number } -> read a text file  
- search_files { pattern: string, path?: string, fileTypes?: ["go","js","py"], caseSensitive?: boolean, maxResults?: number } -> search for text patterns in files using regex
- write_file { path: string, content: string, append?: boolean, reason?: string } -> write or append content to a file (requires approval)
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

// RunAgentTask executes an agent task with retry mechanisms and enhanced functionality
func RunAgentTask(task string, provider providers.LLMProvider, policyStore *policy.Store, verbose bool) error {
	return RunAgentTaskWithWorkingDir(task, provider, policyStore, "", verbose)
}

// RunAgentTaskWithWorkingDir executes an agent task with a specific working directory
func RunAgentTaskWithWorkingDir(task string, provider providers.LLMProvider, policyStore *policy.Store, workingDir string, verbose bool) error {
	maxIters := 60
	transcript := []providers.ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Task: %s\nOS: Windows", task)},
	}

	// Set working directory
	if workingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workingDir = wd
		} else {
			workingDir = "."
		}
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

		// Get LLM response with retry mechanism
		var raw string
		var err error

		for retryCount := 0; retryCount <= maxAPIRetries; retryCount++ {
			raw, err = provider.Chat(transcript, nil)
			if err == nil {
				break // Success, exit retry loop
			}

			if !isRetryableError(err) {
				// Non-retryable error, exit immediately
				break
			}

			if retryCount < maxAPIRetries {
				// Show discrete retry message only if we're going to retry
				if verbose {
					fmt.Printf("  ⎿  API temporarily unavailable, retrying in %v... (%d/%d)\n",
						retryDelay*time.Duration(retryCount+1), retryCount+1, maxAPIRetries)
				}
				time.Sleep(retryDelay * time.Duration(retryCount+1)) // Exponential backoff
			}
		}

		if err != nil {
			if isRetryableError(err) {
				fmt.Printf("● API service temporarily unavailable after %d retries\n", maxAPIRetries)
				fmt.Printf("  ⎿  The LLM provider is experiencing high load. Please try again in a few minutes.\n")
			} else {
				fmt.Printf("● Failed to communicate with LLM provider\n")
				fmt.Printf("  ⎿  %v\n", err)
			}
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
			if err := handleListFiles(action, &transcript, workingDir, verbose); err != nil {
				return err
			}

		case "read_file":
			if err := handleReadFile(action, &transcript, workingDir, verbose); err != nil {
				return err
			}

		case "search_files":
			if err := handleSearchFiles(action, &transcript, workingDir, verbose); err != nil {
				return err
			}

		case "shell":
			if err := handleShell(action, &transcript, policyStore, workingDir, verbose); err != nil {
				return err
			}

		case "write_file":
			if err := handleWriteFile(action, &transcript, policyStore, workingDir, verbose); err != nil {
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
