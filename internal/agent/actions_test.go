package agent

import (
	"testing"
)

func TestAgentAction(t *testing.T) {
	action := &AgentAction{
		Type:    "shell",
		Command: "echo hello",
		Shell:   "powershell",
		Reason:  "test command",
	}

	if action.Type != "shell" {
		t.Errorf("Expected Type to be 'shell', got %q", action.Type)
	}
	if action.Command != "echo hello" {
		t.Errorf("Expected Command to be 'echo hello', got %q", action.Command)
	}
}

func TestSearchResult(t *testing.T) {
	result := &SearchResult{
		File:    "test.go",
		LineNum: 42,
		Line:    "func TestSomething() {",
	}

	if result.File != "test.go" {
		t.Errorf("Expected File to be 'test.go', got %q", result.File)
	}
	if result.LineNum != 42 {
		t.Errorf("Expected LineNum to be 42, got %d", result.LineNum)
	}
	if result.Line != "func TestSomething() {" {
		t.Errorf("Expected Line to be 'func TestSomething() {', got %q", result.Line)
	}
}

func TestParseAgentActionValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple shell action",
			`{"type": "shell", "command": "echo hello", "shell": "powershell"}`,
			"shell",
		},
		{
			"list files action",
			`{"type": "list_files", "path": "."}`,
			"list_files",
		},
		{
			"read file action",
			`{"type": "read_file", "path": "test.go"}`,
			"read_file",
		},
		{
			"done action",
			`{"type": "done", "result": "completed"}`,
			"done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := parseAgentAction(tt.input)
			if err != nil {
				t.Fatalf("parseAgentAction failed: %v", err)
			}
			if action.Type != tt.expected {
				t.Errorf("Expected action type %q, got %q", tt.expected, action.Type)
			}
		})
	}
}

func TestParseAgentActionWithCodeBlock(t *testing.T) {
	input := "```json\n{\"type\": \"shell\", \"command\": \"ls\", \"shell\": \"bash\"}\n```"
	action, err := parseAgentAction(input)
	if err != nil {
		t.Fatalf("parseAgentAction failed: %v", err)
	}
	if action.Type != "shell" {
		t.Errorf("Expected action type 'shell', got %q", action.Type)
	}
	if action.Command != "ls" {
		t.Errorf("Expected command 'ls', got %q", action.Command)
	}
}

func TestParseAgentActionInvalidJSON(t *testing.T) {
	tests := []string{
		`{"type": "invalid"}`,
		`not json at all`,
		`{"command": "echo"}`, // missing required fields
		``,
	}

	for _, input := range tests {
		t.Run("invalid_"+input[:min(len(input), 10)], func(t *testing.T) {
			_, err := parseAgentAction(input)
			if err == nil {
				t.Errorf("Expected parseAgentAction to fail for input %q", input)
			}
		})
	}
}

func TestCoerceActionType(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			"list-files to list_files",
			map[string]interface{}{"type": "list-files"},
			"list_files",
		},
		{
			"get-file to read_file",
			map[string]interface{}{"type": "get-file"},
			"read_file",
		},
		{
			"run-command to shell",
			map[string]interface{}{"type": "run-command"},
			"shell",
		},
		{
			"result to done",
			map[string]interface{}{"type": "result", "text": "completed"},
			"done",
		},
		{
			"command heuristic",
			map[string]interface{}{"command": "echo hello"},
			"shell",
		},
		{
			"result heuristic",
			map[string]interface{}{"result": "done"},
			"done",
		},
		{
			"file path heuristic",
			map[string]interface{}{"path": "test.go"},
			"read_file",
		},
		{
			"directory path heuristic",
			map[string]interface{}{"path": "./src"},
			"list_files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coerceActionType(tt.input)
			if result["type"] != tt.expected {
				t.Errorf("Expected type %q, got %q", tt.expected, result["type"])
			}
		})
	}
}

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name        string
		action      *AgentAction
		expectError bool
	}{
		{
			"valid shell action",
			&AgentAction{Type: "shell", Command: "echo hello"},
			false,
		},
		{
			"shell action missing command",
			&AgentAction{Type: "shell"},
			true,
		},
		{
			"shell action invalid shell",
			&AgentAction{Type: "shell", Command: "echo hello", Shell: "invalid"},
			true,
		},
		{
			"valid read_file action",
			&AgentAction{Type: "read_file", Path: "test.go"},
			false,
		},
		{
			"read_file action missing path",
			&AgentAction{Type: "read_file"},
			true,
		},
		{
			"read_file action invalid maxBytes",
			&AgentAction{Type: "read_file", Path: "test.go", MaxBytes: intPtr(300000)},
			true,
		},
		{
			"valid list_files action",
			&AgentAction{Type: "list_files"},
			false,
		},
		{
			"list_files action invalid depth",
			&AgentAction{Type: "list_files", Depth: intPtr(5)},
			true,
		},
		{
			"valid search_files action",
			&AgentAction{Type: "search_files", Pattern: "test"},
			false,
		},
		{
			"search_files action missing pattern",
			&AgentAction{Type: "search_files"},
			true,
		},
		{
			"valid done action",
			&AgentAction{Type: "done"},
			false,
		},
		{
			"unknown action type",
			&AgentAction{Type: "unknown"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(tt.action)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for action %+v", tt.action)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for action %+v: %v", tt.action, err)
			}
		})
	}
}

func TestValidateActionDefaults(t *testing.T) {
	// Test that validateAction sets proper defaults
	action := &AgentAction{Type: "list_files"}
	err := validateAction(action)
	if err != nil {
		t.Fatalf("validateAction failed: %v", err)
	}

	if action.Path != "." {
		t.Errorf("Expected default path to be '.', got %q", action.Path)
	}
	if action.Depth == nil || *action.Depth != 0 {
		t.Errorf("Expected default depth to be 0, got %v", action.Depth)
	}

	// Test shell defaults
	shellAction := &AgentAction{Type: "shell", Command: "echo hello"}
	err = validateAction(shellAction)
	if err != nil {
		t.Fatalf("validateAction failed: %v", err)
	}

	if shellAction.Shell != "powershell" {
		t.Errorf("Expected default shell to be 'powershell', got %q", shellAction.Shell)
	}

	// Test read_file defaults
	readAction := &AgentAction{Type: "read_file", Path: "test.go"}
	err = validateAction(readAction)
	if err != nil {
		t.Fatalf("validateAction failed: %v", err)
	}

	if readAction.MaxBytes == nil || *readAction.MaxBytes != 4000 {
		t.Errorf("Expected default maxBytes to be 4000, got %v", readAction.MaxBytes)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
