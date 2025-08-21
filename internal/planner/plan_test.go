package planner

import (
	"errors"
	"terminusai/internal/providers"
	"testing"
)

// MockProvider implements the LLMProvider interface for testing
type MockProvider struct {
	response string
	err      error
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) DefaultModel() string {
	return "mock-model"
}

func (m *MockProvider) Chat(messages []providers.ChatMessage, options *providers.ChatOptions) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestPlanStep(t *testing.T) {
	step := PlanStep{
		Description:      "Test step",
		Shell:            "powershell",
		Command:          "echo hello",
		AllowInteractive: false,
		SessionMutation:  false,
		CWD:              "/test",
	}

	if step.Description != "Test step" {
		t.Errorf("Expected Description to be 'Test step', got %q", step.Description)
	}
	if step.Shell != "powershell" {
		t.Errorf("Expected Shell to be 'powershell', got %q", step.Shell)
	}
	if step.Command != "echo hello" {
		t.Errorf("Expected Command to be 'echo hello', got %q", step.Command)
	}
}

func TestPlan(t *testing.T) {
	plan := Plan{
		Steps: []PlanStep{
			{
				Description: "First step",
				Shell:       "powershell",
				Command:     "echo first",
			},
			{
				Description: "Second step",
				Shell:       "powershell",
				Command:     "echo second",
			},
		},
	}

	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Description != "First step" {
		t.Errorf("Expected first step description to be 'First step', got %q", plan.Steps[0].Description)
	}
}

func TestValidatePlan(t *testing.T) {
	tests := []struct {
		name        string
		plan        *Plan
		expectError bool
		errorMsg    string
	}{
		{
			"valid plan",
			&Plan{
				Steps: []PlanStep{
					{
						Description: "Test step",
						Shell:       "powershell",
						Command:     "echo hello",
					},
				},
			},
			false,
			"",
		},
		{
			"nil plan",
			nil,
			true,
			"plan is nil",
		},
		{
			"empty plan",
			&Plan{Steps: []PlanStep{}},
			true,
			"plan has no steps",
		},
		{
			"step missing description",
			&Plan{
				Steps: []PlanStep{
					{
						Shell:   "powershell",
						Command: "echo hello",
					},
				},
			},
			true,
			"step 0: description is required",
		},
		{
			"step missing command",
			&Plan{
				Steps: []PlanStep{
					{
						Description: "Test step",
						Shell:       "powershell",
					},
				},
			},
			true,
			"step 0: command is required",
		},
		{
			"invalid shell",
			&Plan{
				Steps: []PlanStep{
					{
						Description: "Test step",
						Shell:       "invalid",
						Command:     "echo hello",
					},
				},
			},
			true,
			"step 0: shell must be powershell, bash, or cmd",
		},
		{
			"valid bash shell",
			&Plan{
				Steps: []PlanStep{
					{
						Description: "Test step",
						Shell:       "bash",
						Command:     "echo hello",
					},
				},
			},
			false,
			"",
		},
		{
			"valid cmd shell",
			&Plan{
				Steps: []PlanStep{
					{
						Description: "Test step",
						Shell:       "cmd",
						Command:     "echo hello",
					},
				},
			},
			false,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlan(tt.plan)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPlanCommandsSuccess(t *testing.T) {
	validPlanJSON := `{
		"steps": [
			{
				"description": "List files",
				"shell": "powershell",
				"command": "Get-ChildItem",
				"allowInteractive": false,
				"sessionMutation": false
			}
		]
	}`

	provider := &MockProvider{response: validPlanJSON}
	plan, err := PlanCommands("list files", provider)

	if err != nil {
		t.Fatalf("PlanCommands failed: %v", err)
	}
	if plan == nil {
		t.Fatalf("Expected plan to be non-nil")
	}
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Description != "List files" {
		t.Errorf("Expected description 'List files', got %q", plan.Steps[0].Description)
	}
}

func TestPlanCommandsProviderError(t *testing.T) {
	provider := &MockProvider{err: errors.New("API error")}
	_, err := PlanCommands("test task", provider)

	if err == nil {
		t.Fatalf("Expected error but got none")
	}
	if !contains(err.Error(), "failed to get response from provider") {
		t.Errorf("Expected provider error, got %q", err.Error())
	}
}

func TestPlanCommandsInvalidJSON(t *testing.T) {
	provider := &MockProvider{response: "not json"}
	_, err := PlanCommands("test task", provider)

	if err == nil {
		t.Fatalf("Expected error but got none")
	}
	if !contains(err.Error(), "planner returned no JSON") {
		t.Errorf("Expected JSON parsing error, got %q", err.Error())
	}
}

func TestPlanCommandsInvalidPlan(t *testing.T) {
	invalidPlanJSON := `{
		"steps": [
			{
				"description": "",
				"shell": "powershell",
				"command": "echo hello"
			}
		]
	}`

	provider := &MockProvider{response: invalidPlanJSON}
	_, err := PlanCommands("test task", provider)

	if err == nil {
		t.Fatalf("Expected error but got none")
	}
	if !contains(err.Error(), "invalid plan") {
		t.Errorf("Expected validation error, got %q", err.Error())
	}
}

func TestSystemPrompt(t *testing.T) {
	if SystemPrompt == "" {
		t.Errorf("SystemPrompt should not be empty")
	}
	if !contains(SystemPrompt, "CLI command planner") {
		t.Errorf("SystemPrompt should mention CLI command planner")
	}
	if !contains(SystemPrompt, "PowerShell") {
		t.Errorf("SystemPrompt should mention PowerShell")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > len(substr) && (s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexof(s, substr) >= 0)))
}

func indexof(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
