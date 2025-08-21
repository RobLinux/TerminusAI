package planner

import (
	"encoding/json"
	"fmt"
	"regexp"

	"terminusai/internal/config"
	"terminusai/internal/providers"
)

type PlanStep struct {
	Description      string `json:"description"`
	Shell           string `json:"shell"` // powershell, bash, cmd
	Command         string `json:"command"`
	AllowInteractive bool   `json:"allowInteractive"`
	SessionMutation  bool   `json:"sessionMutation"`
	CWD             string `json:"cwd,omitempty"`
}

type Plan struct {
	Steps []PlanStep `json:"steps"`
}

const SystemPrompt = `You are a CLI command planner. Given a user task, respond ONLY with JSON that matches this schema:
{
  "steps": [
    { "description": string, "shell": "powershell"|"bash"|"cmd", "command": string, "allowInteractive": boolean, "sessionMutation": boolean, "cwd"?: string }
  ]
}
Rules:
- Prefer PowerShell on Windows.
- Use sessionMutation=true for cd, set env vars, activate venv, etc.
- Use concise, safe commands. Avoid destructive actions unless explicitly asked.
- Set cwd when commands must run in a subfolder.
- Keep steps minimal and deterministic.`

func PlanCommands(task string, provider providers.LLMProvider) (*Plan, error) {
	messages := []providers.ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Task: %s\nOS: Windows", task)},
	}

	raw, err := provider.Chat(messages, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from provider: %w", err)
	}

	cm := config.GetConfigManager()
	debug := cm.IsDebug()
	if debug {
		fmt.Println("[llm] planner raw ->")
		fmt.Println(raw)
	}

	// Find first JSON block
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	match := jsonRegex.FindString(raw)
	if match == "" {
		return nil, fmt.Errorf("planner returned no JSON")
	}

	var plan Plan
	if err := json.Unmarshal([]byte(match), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the plan
	if err := validatePlan(&plan); err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	if debug {
		fmt.Println("[llm] planner plan ->")
		planJSON, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Println(string(planJSON))
	}

	return &plan, nil
}

func validatePlan(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}

	if len(plan.Steps) == 0 {
		return fmt.Errorf("plan has no steps")
	}

	for i, step := range plan.Steps {
		if step.Description == "" {
			return fmt.Errorf("step %d: description is required", i)
		}
		if step.Command == "" {
			return fmt.Errorf("step %d: command is required", i)
		}
		if step.Shell != "powershell" && step.Shell != "bash" && step.Shell != "cmd" {
			return fmt.Errorf("step %d: shell must be powershell, bash, or cmd", i)
		}
	}

	return nil
}

