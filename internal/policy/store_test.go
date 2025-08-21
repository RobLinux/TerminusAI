package policy

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDecisionConstants(t *testing.T) {
	tests := []struct {
		name     string
		decision Decision
		expected string
	}{
		{"DecisionOnce", DecisionOnce, "once"},
		{"DecisionAlways", DecisionAlways, "always"},
		{"DecisionNever", DecisionNever, "never"},
		{"DecisionSkip", DecisionSkip, "skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.decision) != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, string(tt.decision))
			}
		})
	}
}

func TestRule(t *testing.T) {
	rule := Rule{
		Pattern:  "echo *",
		Decision: DecisionAlways,
	}

	if rule.Pattern != "echo *" {
		t.Errorf("Expected Pattern to be 'echo *', got %q", rule.Pattern)
	}
	if rule.Decision != DecisionAlways {
		t.Errorf("Expected Decision to be DecisionAlways, got %q", rule.Decision)
	}
}

func TestStoreSetAlwaysAllow(t *testing.T) {
	store := &Store{}

	// Test default state
	if store.IsAlwaysAllow() {
		t.Errorf("Expected alwaysAllow to be false by default")
	}

	// Test enabling
	store.SetAlwaysAllow(true)
	if !store.IsAlwaysAllow() {
		t.Errorf("Expected alwaysAllow to be true after enabling")
	}

	// Test disabling
	store.SetAlwaysAllow(false)
	if store.IsAlwaysAllow() {
		t.Errorf("Expected alwaysAllow to be false after disabling")
	}
}

func TestStoreFind(t *testing.T) {
	store := &Store{
		rules: []Rule{
			{Pattern: "echo *", Decision: DecisionAlways},
			{Pattern: "ls", Decision: DecisionNever},
		},
	}

	// Test finding existing rule
	rule := store.Find("echo *")
	if rule == nil {
		t.Fatalf("Expected to find rule 'echo *'")
	}
	if rule.Decision != DecisionAlways {
		t.Errorf("Expected decision to be DecisionAlways, got %q", rule.Decision)
	}

	// Test finding non-existing rule
	rule = store.Find("rm *")
	if rule != nil {
		t.Errorf("Expected not to find rule 'rm *', but found %+v", rule)
	}
}

func TestStoreAdd(t *testing.T) {
	store := &Store{
		rules: []Rule{
			{Pattern: "echo *", Decision: DecisionAlways},
		},
	}

	// Test adding new rule
	newRule := Rule{Pattern: "ls", Decision: DecisionNever}
	store.Add(newRule)

	if len(store.rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(store.rules))
	}

	found := store.Find("ls")
	if found == nil {
		t.Fatalf("Expected to find newly added rule")
	}
	if found.Decision != DecisionNever {
		t.Errorf("Expected decision to be DecisionNever, got %q", found.Decision)
	}

	// Test updating existing rule
	updateRule := Rule{Pattern: "echo *", Decision: DecisionNever}
	store.Add(updateRule)

	if len(store.rules) != 2 {
		t.Errorf("Expected still 2 rules after update, got %d", len(store.rules))
	}

	found = store.Find("echo *")
	if found == nil {
		t.Fatalf("Expected to find updated rule")
	}
	if found.Decision != DecisionNever {
		t.Errorf("Expected decision to be updated to DecisionNever, got %q", found.Decision)
	}
}

func TestStoreApproveWithAlwaysAllow(t *testing.T) {
	store := &Store{}
	store.SetAlwaysAllow(true)

	decision, err := store.Approve("any command", "test description")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if decision != DecisionAlways {
		t.Errorf("Expected DecisionAlways when alwaysAllow is true, got %q", decision)
	}
}

func TestStoreApproveWithExistingRule(t *testing.T) {
	store := &Store{
		rules: []Rule{
			{Pattern: "echo hello", Decision: DecisionAlways},
			{Pattern: "rm *", Decision: DecisionNever},
		},
	}

	// Test exact match
	decision, err := store.Approve("echo hello", "test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if decision != DecisionAlways {
		t.Errorf("Expected DecisionAlways for 'echo hello', got %q", decision)
	}

	// Test wildcard match
	decision, err = store.Approve("rm file.txt", "test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if decision != DecisionNever {
		t.Errorf("Expected DecisionNever for 'rm file.txt', got %q", decision)
	}
}

func TestLoadStoreWithExistingFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "terminusai_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test policy file
	policyFile := filepath.Join(tmpDir, "policy.json")
	policyContent := `[
		{"pattern": "echo *", "decision": "always"},
		{"pattern": "rm *", "decision": "never"}
	]`

	err = ioutil.WriteFile(policyFile, []byte(policyContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test policy file: %v", err)
	}

	// Mock the home directory to point to our temp dir
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	// Set HOME to a subdirectory so Load() will use our test directory
	testHome := filepath.Join(tmpDir, "home")
	os.MkdirAll(filepath.Join(testHome, ".terminusai"), 0755)
	err = ioutil.WriteFile(filepath.Join(testHome, ".terminusai", "policy.json"), []byte(policyContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test policy file: %v", err)
	}

	if runtime := os.Getenv("HOME"); runtime != "" {
		os.Setenv("HOME", testHome)
		defer os.Setenv("HOME", originalHome)
	} else {
		os.Setenv("USERPROFILE", testHome)
		defer os.Setenv("USERPROFILE", originalHome)
	}

	// Test loading
	store, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(store.rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(store.rules))
	}

	rule := store.Find("echo *")
	if rule == nil {
		t.Fatalf("Expected to find 'echo *' rule")
	}
	if rule.Decision != DecisionAlways {
		t.Errorf("Expected DecisionAlways, got %q", rule.Decision)
	}
}

func TestLoadStoreWithNonExistentFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "terminusai_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock the home directory
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	if runtime := os.Getenv("HOME"); runtime != "" {
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", originalHome)
	} else {
		os.Setenv("USERPROFILE", tmpDir)
		defer os.Setenv("USERPROFILE", originalHome)
	}

	// Test loading when file doesn't exist
	store, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(store.rules) != 0 {
		t.Errorf("Expected 0 rules for new store, got %d", len(store.rules))
	}
}

func TestStoreSave(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "terminusai_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	policyFile := filepath.Join(tmpDir, "policy.json")

	store := &Store{
		rules: []Rule{
			{Pattern: "echo *", Decision: DecisionAlways},
			{Pattern: "rm *", Decision: DecisionNever},
		},
		file: policyFile,
	}

	err = store.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify the file was created and has correct content
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		t.Fatalf("Policy file was not created")
	}

	content, err := ioutil.ReadFile(policyFile)
	if err != nil {
		t.Fatalf("Failed to read saved policy file: %v", err)
	}

	expected := `[
  {
    "pattern": "echo *",
    "decision": "always"
  },
  {
    "pattern": "rm *",
    "decision": "never"
  }
]`

	if string(content) != expected {
		t.Errorf("Saved content doesn't match expected.\nExpected:\n%s\nGot:\n%s", expected, string(content))
	}
}
