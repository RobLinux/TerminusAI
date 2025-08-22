package agent

import (
	"fmt"
	"time"

	"terminusai/internal/policy"
	"terminusai/internal/providers"
	"terminusai/internal/ui"
)

// RunAgentTask executes an agent task with retry mechanisms and enhanced functionality
func RunAgentTask(task string, provider providers.LLMProvider, policyStore *policy.Store, verbose bool) error {
	return RunAgentTaskWithWorkingDir(task, provider, policyStore, "", verbose)
}

// RunAgentTaskWithWorkingDir executes an agent task with a specific working directory
func RunAgentTaskWithWorkingDir(task string, provider providers.LLMProvider, policyStore *policy.Store, workingDir string, verbose bool) error {
	// Use the new unified Agent instead of the old basic implementation
	agent := NewAgent(provider, policyStore, workingDir, verbose, false)
	return agent.RunTask(task)
}

// RunTask executes a task with UI feedback
func (a *Agent) RunTask(task string) error {
	// Show thinking phase
	spinner := a.display.ShowAgentThinking(task)

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

		// API retry logic with exponential backoff
		maxAPIRetries := 3
		retryDelay := 500 * time.Millisecond
		var raw string
		var err error

		for retryCount := 0; retryCount <= maxAPIRetries; retryCount++ {
			// Log request in debug/verbose mode
			if a.debug || a.verbose {
				fmt.Printf("\n🔄 LLM Request (attempt %d/%d):\n", retryCount+1, maxAPIRetries+1)
				for i, msg := range transcript {
					fmt.Printf("  [%d] %s: %s\n", i, msg.Role, truncateString(msg.Content, 200))
				}
				fmt.Printf("\n")
			}

			raw, err = a.provider.Chat(transcript, nil)

			// Log response in debug/verbose mode
			if a.debug || a.verbose {
				if err != nil {
					fmt.Printf("🚨 LLM Error: %v\n\n", err)
				} else {
					fmt.Printf("✅ LLM Response: %s\n\n", truncateString(raw, 300))
				}
			}

			if err == nil {
				break
			}

			if !isRetryableError(err) {
				// Non-retryable error, exit immediately
				break
			}

			if retryCount < maxAPIRetries {
				// Show discrete retry message only if we're going to retry
				if a.verbose {
					fmt.Printf("  ⎿  API temporarily unavailable, retrying in %v... (%d/%d)\n",
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

		// Execute action
		switch action.Type {
		case "done":
			result := action.Result
			if result == "" {
				result = "Task completed successfully"
			}

			// Just show simple completion message
			fmt.Println()
			ui.Success.Printf("✓ %s\n", result)
			a.display.ShowAgentSummary()
			return nil

		case "list_files":
			if err := a.handleListFiles(action, &transcript); err != nil {
				return err
			}

		case "read_file":
			if err := a.handleReadFile(action, &transcript); err != nil {
				return err
			}

		case "search_files":
			if err := a.handleSearchFiles(action, &transcript); err != nil {
				return err
			}

		case "shell":
			if err := a.handleShell(action, &transcript); err != nil {
				return err
			}

		case "write_file":
			if err := a.handleWriteFile(action, &transcript); err != nil {
				return err
			}

		case "ps":
			if err := a.handlePs(action, &transcript); err != nil {
				return err
			}

		case "kill":
			if err := a.handleKill(action, &transcript); err != nil {
				return err
			}

		case "http_request":
			if err := a.handleHttpRequest(action, &transcript); err != nil {
				return err
			}

		case "ping":
			if err := a.handlePing(action, &transcript); err != nil {
				return err
			}

		case "traceroute":
			if err := a.handleTraceroute(action, &transcript); err != nil {
				return err
			}

		case "get_system_info":
			if err := a.handleGetSystemInfo(action, &transcript); err != nil {
				return err
			}

		case "install_package":
			if err := a.handleInstallPackage(action, &transcript); err != nil {
				return err
			}

		case "git":
			if err := a.handleGit(action, &transcript); err != nil {
				return err
			}

		case "extract":
			if err := a.handleExtract(action, &transcript); err != nil {
				return err
			}

		case "compress":
			if err := a.handleCompress(action, &transcript); err != nil {
				return err
			}

		case "parse_json":
			if err := a.handleParseJson(action, &transcript); err != nil {
				return err
			}

		case "parse_yaml":
			if err := a.handleParseYaml(action, &transcript); err != nil {
				return err
			}

		case "ask_user":
			if err := a.handleAskUser(action, &transcript); err != nil {
				return err
			}

		case "log":
			if err := a.handleLog(action, &transcript); err != nil {
				return err
			}

		case "copy_path":
			if err := a.handleCopyPath(action, &transcript); err != nil {
				return err
			}

		case "move_path":
			if err := a.handleMovePath(action, &transcript); err != nil {
				return err
			}

		case "delete_path":
			if err := a.handleDeletePath(action, &transcript); err != nil {
				return err
			}

		case "stat_path":
			if err := a.handleStatPath(action, &transcript); err != nil {
				return err
			}

		case "make_dir":
			if err := a.handleMakeDir(action, &transcript); err != nil {
				return err
			}

		case "patch_file":
			if err := a.handlePatchFile(action, &transcript); err != nil {
				return err
			}

		case "download_file":
			if err := a.handleDownloadFile(action, &transcript); err != nil {
				return err
			}

		case "grep":
			if err := a.handleGrep(action, &transcript); err != nil {
				return err
			}

		case "diff":
			if err := a.handleDiff(action, &transcript); err != nil {
				return err
			}

		case "parse":
			if err := a.handleParse(action, &transcript); err != nil {
				return err
			}

		case "confirm":
			if err := a.handleConfirm(action, &transcript); err != nil {
				return err
			}

		case "report":
			if err := a.handleReport(action, &transcript); err != nil {
				return err
			}

		case "uuid":
			if err := a.handleUuid(action, &transcript); err != nil {
				return err
			}

		case "time_now":
			if err := a.handleTimeNow(action, &transcript); err != nil {
				return err
			}

		case "hash_file":
			if err := a.handleHashFile(action, &transcript); err != nil {
				return err
			}

		case "checksum_verify":
			if err := a.handleChecksumVerify(action, &transcript); err != nil {
				return err
			}

		case "hexdump":
			if err := a.handleHexdump(action, &transcript); err != nil {
				return err
			}

		case "env_get":
			if err := a.handleEnvGet(action, &transcript); err != nil {
				return err
			}

		case "env_set":
			if err := a.handleEnvSet(action, &transcript); err != nil {
				return err
			}

		case "whoami":
			if err := a.handleWhoami(action, &transcript); err != nil {
				return err
			}

		default:
			errorMsg := fmt.Sprintf("Unknown action type: %s", action.Type)
			transcript = append(transcript, providers.ChatMessage{Role: "user", Content: errorMsg})
		}
	}

	a.display.ShowAction("Max iterations reached", "Agent stopped after reaching maximum iterations", false)
	a.display.ShowAgentSummary()
	return nil
}
