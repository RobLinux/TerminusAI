package agent

import (
	"os"

	"terminusai/internal/policy"
	"terminusai/internal/providers"
	"terminusai/internal/ui"
)

// Agent provides an agent with UI and interactivity
type Agent struct {
	provider    providers.LLMProvider
	policyStore *policy.Store
	display     *ui.InteractiveDisplay
	workingDir  string
	verbose     bool
	debug       bool
}

// NewAgent creates a new agent
func NewAgent(provider providers.LLMProvider, policyStore *policy.Store, workingDir string, verbose, debug bool) *Agent {
	if workingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workingDir = wd
		} else {
			workingDir = "."
		}
	}

	return &Agent{
		provider:    provider,
		policyStore: policyStore,
		display:     ui.NewInteractiveDisplay(verbose, debug),
		workingDir:  workingDir,
		verbose:     verbose,
		debug:       debug,
	}
}