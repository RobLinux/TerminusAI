package planner

import (
	"encoding/json"
	"fmt"
	"regexp"

	"terminusai/internal/config"
	"terminusai/internal/providers"
	"terminusai/internal/ui"
)

type PlanStep struct {
	Description      string `json:"description"`
	Shell            string `json:"shell"` // powershell, bash, cmd
	Command          string `json:"command"`
	AllowInteractive bool   `json:"allowInteractive"`
	SessionMutation  bool   `json:"sessionMutation"`
	CWD              string `json:"cwd,omitempty"`
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

// Planner provides planning with visual feedback
type Planner struct {
	display *ui.Display
}

// NewPlanner creates a new planner
func NewPlanner(verbose, debug bool) *Planner {
	return &Planner{
		display: ui.NewDisplay(verbose, debug),
	}
}

// PlanCommandsWithUI plans commands with visual synthesis feedback
func (p *Planner) PlanCommandsWithUI(task string, provider providers.LLMProvider) (*Plan, error) {
	// Show synthesis start
	spinner := p.display.PrintSynthesis(task)

	// Prepare messages for the LLM
	messages := []providers.ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Task: %s\nOS: Windows", task)},
	}

	p.display.PrintDebug("Sending request to LLM provider")

	// Get response from provider
	raw, err := provider.Chat(messages, nil)
	spinner.Stop() // Stop spinner before showing result

	if err != nil {
		p.display.PrintError("Failed to get response from provider: %v", err)
		return nil, fmt.Errorf("failed to get response from provider: %w", err)
	}

	p.display.PrintDebug("Received response, parsing...")

	// Get configuration manager for debug output
	cm := config.GetConfigManager()
	debug := cm.IsDebug()

	if debug {
		p.display.PrintDebug("Raw LLM response:")
		fmt.Println(raw)
	}

	// Parse the JSON response
	plan, err := p.parseResponse(raw)
	if err != nil {
		p.display.PrintError("Failed to parse response: %v", err)
		return nil, err
	}

	// Validate the plan
	if err := validatePlan(plan); err != nil {
		p.display.PrintError("Invalid plan generated: %v", err)
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	// Show successful plan generation
	p.display.PrintPlan(len(plan.Steps))

	if debug {
		p.display.PrintDebug("Generated plan:")
		planJSON, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Println(string(planJSON))
	}

	// Display plan preview
	p.displayPlanPreview(plan)

	return plan, nil
}

// parseResponse extracts and parses the JSON from the LLM response
func (p *Planner) parseResponse(raw string) (*Plan, error) {
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

	return &plan, nil
}

// displayPlanPreview shows a formatted preview of the execution plan
func (p *Planner) displayPlanPreview(plan *Plan) {
	if len(plan.Steps) == 0 {
		return
	}

	p.display.PrintSection("Execution Plan Preview")

	for i, step := range plan.Steps {
		p.display.PrintTask(fmt.Sprintf("Step %d: %s", i+1, step.Description))
		p.display.PrintCommand(step.Shell, step.Command)

		// Show additional details in verbose mode
		p.display.PrintVerbose("  Interactive: %t, Session Mutation: %t",
			step.AllowInteractive, step.SessionMutation)

		if step.CWD != "" {
			p.display.PrintVerbose("  Working Directory: %s", step.CWD)
		}
	}
}
