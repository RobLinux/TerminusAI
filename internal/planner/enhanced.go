package planner

import (
	"encoding/json"
	"fmt"
	"regexp"

	"terminusai/internal/config"
	"terminusai/internal/providers"
	"terminusai/internal/ui"
)

// EnhancedPlanner provides planning with visual feedback
type EnhancedPlanner struct {
	display *ui.Display
}

// NewEnhancedPlanner creates a new enhanced planner
func NewEnhancedPlanner(verbose, debug bool) *EnhancedPlanner {
	return &EnhancedPlanner{
		display: ui.NewDisplay(verbose, debug),
	}
}

// PlanCommandsWithUI plans commands with visual synthesis feedback
func (p *EnhancedPlanner) PlanCommandsWithUI(task string, provider providers.LLMProvider) (*Plan, error) {
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
func (p *EnhancedPlanner) parseResponse(raw string) (*Plan, error) {
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
func (p *EnhancedPlanner) displayPlanPreview(plan *Plan) {
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