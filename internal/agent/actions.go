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
	// Write file fields
	Content string `json:"content,omitempty"`
	Append  *bool  `json:"append,omitempty"`
	// Process management fields
	ProcessID *int `json:"processId,omitempty"`
	// Network fields
	Method  string            `json:"method,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Host    string            `json:"host,omitempty"`
	// Package management fields
	Name    string `json:"name,omitempty"`
	Manager string `json:"manager,omitempty"`
	// Archive fields
	ArchivePath string   `json:"archivePath,omitempty"`
	Files       []string `json:"files,omitempty"`
	// Interactive fields
	Question string `json:"question,omitempty"`
	// Logging fields
	Level   string `json:"level,omitempty"`
	Message string `json:"message,omitempty"`
	// File system operation fields
	Src         string `json:"src,omitempty"`
	Dest        string `json:"dest,omitempty"`
	Overwrite   *bool  `json:"overwrite,omitempty"`
	Recursive   *bool  `json:"recursive,omitempty"`
	Parents     *bool  `json:"parents,omitempty"`
	// Patch fields
	Patch  string `json:"patch,omitempty"`
	Format string `json:"format,omitempty"`
	// Diff fields
	APath   string `json:"aPath,omitempty"`
	BPath   string `json:"bPath,omitempty"`
	Context *int   `json:"context,omitempty"`
	// Parse fields
	ParseType string `json:"parseType,omitempty"`
	// Enhanced user interaction fields
	Rationale   string      `json:"rationale,omitempty"`
	ActionName  string      `json:"action,omitempty"`
	Details     interface{} `json:"details,omitempty"`
	// Report fields
	Attachments []struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"attachments,omitempty"`
	// Utility fields
	Version   *int   `json:"v,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	TZ        string `json:"tz,omitempty"`
	Algo      string `json:"algo,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
	Offset    *int   `json:"offset,omitempty"`
	// Environment fields
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Persist *bool  `json:"persist,omitempty"`
	// Process fields
	Filter string `json:"filter,omitempty"`
	PID    *int   `json:"pid,omitempty"`
	Signal string `json:"signal,omitempty"`
	// Enhanced search/diff fields
	Regex *bool `json:"regex,omitempty"`
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
	case "write_file":
		if action.Path == "" {
			return fmt.Errorf("path is required for write_file")
		}
		if action.Content == "" {
			return fmt.Errorf("content is required for write_file")
		}
		if action.Append == nil {
			append := false
			action.Append = &append
		}
	case "done":
		if action.Result == "" {
			action.Result = ""
		}
	case "ps":
		// No validation needed for process list
	case "kill":
		if action.ProcessID == nil {
			return fmt.Errorf("processId is required for kill")
		}
	case "http_request":
		if action.URL == "" {
			return fmt.Errorf("url is required for http_request")
		}
		if action.Method == "" {
			action.Method = "GET"
		}
	case "ping":
		if action.Host == "" {
			return fmt.Errorf("host is required for ping")
		}
	case "traceroute":
		if action.Host == "" {
			return fmt.Errorf("host is required for traceroute")
		}
	case "get_system_info":
		// No validation needed for system info
	case "install_package":
		if action.Name == "" {
			return fmt.Errorf("name is required for install_package")
		}
		if action.Manager == "" {
			return fmt.Errorf("manager is required for install_package")
		}
	case "git":
		if action.Command == "" {
			return fmt.Errorf("command is required for git")
		}
	case "extract":
		if action.ArchivePath == "" {
			return fmt.Errorf("archivePath is required for extract")
		}
		if action.Dest == "" {
			return fmt.Errorf("dest is required for extract")
		}
	case "compress":
		if len(action.Files) == 0 {
			return fmt.Errorf("files is required for compress")
		}
		if action.Dest == "" {
			return fmt.Errorf("dest is required for compress")
		}
	case "parse_json":
		if action.Path == "" {
			return fmt.Errorf("path is required for parse_json")
		}
	case "parse_yaml":
		if action.Path == "" {
			return fmt.Errorf("path is required for parse_yaml")
		}
	case "ask_user":
		if action.Question == "" {
			return fmt.Errorf("question is required for ask_user")
		}
	case "log":
		if action.Message == "" {
			return fmt.Errorf("message is required for log")
		}
		if action.Level == "" {
			action.Level = "info"
		}
	case "copy_path":
		if action.Src == "" {
			return fmt.Errorf("src is required for copy_path")
		}
		if action.Dest == "" {
			return fmt.Errorf("dest is required for copy_path")
		}
		if action.Overwrite == nil {
			overwrite := false
			action.Overwrite = &overwrite
		}
	case "move_path":
		if action.Src == "" {
			return fmt.Errorf("src is required for move_path")
		}
		if action.Dest == "" {
			return fmt.Errorf("dest is required for move_path")
		}
		if action.Overwrite == nil {
			overwrite := false
			action.Overwrite = &overwrite
		}
	case "delete_path":
		if action.Path == "" {
			return fmt.Errorf("path is required for delete_path")
		}
		if action.Recursive == nil {
			recursive := false
			action.Recursive = &recursive
		}
	case "stat_path":
		if action.Path == "" {
			return fmt.Errorf("path is required for stat_path")
		}
	case "make_dir":
		if action.Path == "" {
			return fmt.Errorf("path is required for make_dir")
		}
		if action.Parents == nil {
			parents := false
			action.Parents = &parents
		}
	case "patch_file":
		if action.Path == "" {
			return fmt.Errorf("path is required for patch_file")
		}
		if action.Patch == "" {
			return fmt.Errorf("patch is required for patch_file")
		}
		if action.Format == "" {
			action.Format = "unified"
		}
	case "download_file":
		if action.URL == "" {
			return fmt.Errorf("url is required for download_file")
		}
		if action.Dest == "" {
			return fmt.Errorf("dest is required for download_file")
		}
	case "grep":
		if action.Pattern == "" {
			return fmt.Errorf("pattern is required for grep")
		}
		if action.Path == "" {
			action.Path = "."
		}
		if action.Regex == nil {
			regex := false
			action.Regex = &regex
		}
		if action.CaseSensitive == nil {
			caseSensitive := false
			action.CaseSensitive = &caseSensitive
		}
		if action.MaxResults == nil {
			maxResults := 50
			action.MaxResults = &maxResults
		}
	case "diff":
		if action.APath == "" {
			return fmt.Errorf("aPath is required for diff")
		}
		if action.BPath == "" {
			return fmt.Errorf("bPath is required for diff")
		}
		if action.Context == nil {
			context := 3
			action.Context = &context
		}
		if action.Format == "" {
			action.Format = "unified"
		}
	case "parse":
		if action.Path == "" {
			return fmt.Errorf("path is required for parse")
		}
		if action.ParseType == "" {
			return fmt.Errorf("parseType is required for parse")
		}
	case "confirm":
		if action.ActionName == "" {
			return fmt.Errorf("action is required for confirm")
		}
	case "report":
		if action.Result == "" {
			return fmt.Errorf("result is required for report")
		}
	case "uuid":
		if action.Version == nil {
			version := 4
			action.Version = &version
		}
	case "time_now":
		// No validation needed
	case "hash_file":
		if action.Path == "" {
			return fmt.Errorf("path is required for hash_file")
		}
		if action.Algo == "" {
			action.Algo = "sha256"
		}
	case "checksum_verify":
		if action.Path == "" {
			return fmt.Errorf("path is required for checksum_verify")
		}
		if action.Checksum == "" {
			return fmt.Errorf("checksum is required for checksum_verify")
		}
		if action.Algo == "" {
			action.Algo = "sha256"
		}
	case "hexdump":
		if action.Path == "" {
			return fmt.Errorf("path is required for hexdump")
		}
		if action.MaxBytes == nil {
			maxBytes := 1024
			action.MaxBytes = &maxBytes
		}
		if action.Offset == nil {
			offset := 0
			action.Offset = &offset
		}
	case "env_get":
		// key is optional for env_get
	case "env_set":
		if action.Key == "" {
			return fmt.Errorf("key is required for env_set")
		}
		if action.Value == "" {
			return fmt.Errorf("value is required for env_set")
		}
		if action.Persist == nil {
			persist := false
			action.Persist = &persist
		}
	case "whoami":
		// No validation needed
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}

	return nil
}
