package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// AgentAction represents an action that the agent can perform
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
	Pattern       string   `json:"pattern,omitempty"`
	FileTypes     []string `json:"fileTypes,omitempty"`
	CaseSensitive *bool    `json:"caseSensitive,omitempty"`
	MaxResults    *int     `json:"maxResults,omitempty"`
}

// SearchResult represents a search match result
type SearchResult struct {
	File    string
	LineNum int
	Line    string
}

// parseAgentAction parses raw LLM output into an AgentAction
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

// coerceActionType normalizes action type names and provides defaults
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

// validateAction validates and sets defaults for actions
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
