package main

import (
	"fmt"
	"strings"
	"time"

	"terminusai/internal/ui"
)

// Demo script to showcase the enhanced agent UI features
func main() {
	// Enable colors
	ui.EnableColors()
	
	// Create enhanced interactive display
	display := ui.NewInteractiveDisplayEnhanced(false, false)
	display.SetupGlobalShortcuts()
	
	// Demo: Header
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("    TerminusAI Enhanced Agent UI Demo")
	fmt.Println("═══════════════════════════════════════════════")
	
	// Demo: Agent thinking
	fmt.Println("\nStarting agent task: Build this directory into an exe file")
	
	spinner := display.ShowAgentThinking("Build this directory into an exe file")
	time.Sleep(2 * time.Second)
	spinner.Stop()
	
	// Demo: Interactive actions
	fmt.Println("\n")
	
	// List files action
	listAction := display.ShowListFiles(".", 23)
	time.Sleep(1 * time.Second)
	fileList := []string{
		"cmd/",
		"internal/",
		"examples/",
		"go.mod", 
		"go.sum",
		"README.md",
		"UI_ENHANCEMENT.md",
		"terminusai.exe",
		"terminusai-enhanced.exe",
		// ... more files would be here
	}
	display.UpdateAction(listAction, "completed", fileList)
	
	// Simulate user choosing to expand
	fmt.Print("\nShow detailed output? [y/N]: y")
	fmt.Println()
	
	listAction.Details = fileList
	listAction.Output = "Found project structure with Go modules and executables"
	display.ShowExpandedDetails(listAction)
	
	// Read file action  
	readAction := display.ShowReadFile("go.mod", 156)
	time.Sleep(800 * time.Millisecond)
	goModContent := []string{
		"module terminusai",
		"",
		"go 1.21", 
		"",
		"require (",
		"    github.com/fatih/color v1.16.0",
		"    github.com/spf13/cobra v1.8.0",
		"    // ... other dependencies",
		")",
	}
	display.UpdateAction(readAction, "completed", goModContent)
	
	// Shell command action
	shellAction := display.ShowShellCommand("powershell", "go build -o terminusai-built.exe ./cmd/terminusai", "Build Go project into executable")
	time.Sleep(1500 * time.Millisecond)
	
	shellOutput := "Build completed successfully\nExecutable created: terminusai-built.exe"
	shellAction.Output = shellOutput
	display.UpdateAction(shellAction, "completed", []string{"Build completed successfully", "Output: terminusai-built.exe"})
	
	// Final completion
	time.Sleep(500 * time.Millisecond)
	completedAction := display.ShowAction("Task completed", "Successfully built the Go project into terminusai-built.exe executable file", false)
	display.UpdateAction(completedAction, "completed", []string{"Executable created successfully"})
	
	// Show summary
	display.ShowAgentSummary()
	
	// Demo keyboard shortcuts
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Println("Enhanced Features Demonstrated:")
	fmt.Println("● Interactive action display with bullet points")
	fmt.Println("  ⎿  Summary with expandable details (Enter=expand)")
	fmt.Println("● Real-time status updates with timing")
	fmt.Println("● Color-coded completion status")
	fmt.Println("● Detailed output on demand")
	fmt.Println("● Professional formatting without emoji")
	fmt.Println("● Agent execution summary")
	
	display.ShowContextualHelp()
}