package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

type Decision string

const (
	DecisionOnce   Decision = "once"
	DecisionAlways Decision = "always"
	DecisionNever  Decision = "never"
	DecisionSkip   Decision = "skip"
)

type Rule struct {
	Pattern  string   `json:"pattern"`  // simple wildcard * for command matching
	Decision Decision `json:"decision"` // always|never persisted
}

type Store struct {
	rules       []Rule
	file        string
	alwaysAllow bool // Global flag to bypass all approval prompts
}

func Load() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".terminusai")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file := filepath.Join(dir, "policy.json")

	var rules []Rule
	data, err := os.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// File doesn't exist, start with empty rules
		rules = []Rule{}
	} else {
		if err := json.Unmarshal(data, &rules); err != nil {
			return nil, fmt.Errorf("failed to parse policy file: %w", err)
		}
	}

	return &Store{
		rules: rules,
		file:  file,
	}, nil
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.file, data, 0644)
}

func (s *Store) Find(rulePattern string) *Rule {
	for i := range s.rules {
		if s.rules[i].Pattern == rulePattern {
			return &s.rules[i]
		}
	}
	return nil
}

func (s *Store) Add(rule Rule) {
	existing := s.Find(rule.Pattern)
	if existing != nil {
		existing.Decision = rule.Decision
	} else {
		s.rules = append(s.rules, rule)
	}
}

// SetAlwaysAllow enables or disables global always-allow mode
func (s *Store) SetAlwaysAllow(enabled bool) {
	s.alwaysAllow = enabled
}

// IsAlwaysAllow returns whether global always-allow mode is enabled
func (s *Store) IsAlwaysAllow() bool {
	return s.alwaysAllow
}

func (s *Store) Approve(command, description string) (Decision, error) {
	// Check global always-allow mode first
	if s.alwaysAllow {
		return DecisionAlways, nil
	}

	// Check persisted rules
	for _, rule := range s.rules {
		pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(rule.Pattern), "\\*", ".*") + "$"
		matched, err := regexp.MatchString(pattern, command)
		if err != nil {
			continue // Skip malformed patterns
		}
		if matched {
			return rule.Decision, nil
		}
	}

	// Display command information before prompt
	s.displayCommandInfo(command, description)

	// Prompt user for decision
	prompt := promptui.Select{
		Label: "Choose action",
		Items: []string{
			"Allow once",
			"Always allow (persist rule)",
			"Never allow (persist rule)",
			"Skip this command",
		},
	}

	_, result, err := prompt.Run()
	if err != nil {
		return DecisionSkip, err
	}

	var decision Decision
	switch result {
	case "Allow once":
		decision = DecisionOnce
	case "Always allow (persist rule)":
		decision = DecisionAlways
		s.Add(Rule{Pattern: command, Decision: DecisionAlways})
	case "Never allow (persist rule)":
		decision = DecisionNever
		s.Add(Rule{Pattern: command, Decision: DecisionNever})
	default:
		decision = DecisionSkip
	}

	return decision, nil
}

// displayCommandInfo shows command details before approval prompt
func (s *Store) displayCommandInfo(command, description string) {
	// Colors for display
	yellow := color.New(color.FgYellow, color.Bold)
	cyan := color.New(color.FgCyan)
	white := color.New(color.FgWhite, color.Bold)
	muted := color.New(color.FgHiBlack)

	// Show header
	fmt.Println()
	yellow.Println("⚠ Command Approval Required")
	muted.Println(strings.Repeat("─", 50))

	// Show purpose/description
	if description != "" && description != "Run command" {
		cyan.Printf("Purpose: %s\n", description)
	}

	// Show the actual command
	white.Printf("Command: %s\n", command)

	// Show working directory context
	if wd, err := os.Getwd(); err == nil {
		muted.Printf("Working directory: %s\n", wd)
	}

	fmt.Println()
}
