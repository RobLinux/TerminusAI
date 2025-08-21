package main

import (
	"fmt"
	"time"

	"terminusai/internal/ui"
)

// Demo script to showcase the enhanced UI features
func main() {
	// Enable colors
	ui.EnableColors()
	
	// Create display
	display := ui.NewDisplay(true, false)
	
	// Demo: Header
	display.PrintHeader("TerminusAI Enhanced UI Demo")
	
	// Demo: Task display
	display.PrintInfo("Demonstrating enhanced UI capabilities...")
	
	// Demo: Command synthesis with spinner
	fmt.Println()
	spinner := display.PrintSynthesis("create a web server and deploy it")
	time.Sleep(3 * time.Second)
	spinner.Stop()
	
	display.PrintPlan(4)
	
	// Demo: Section and plan preview
	display.PrintSection("Execution Plan Preview")
	
	display.PrintTask("Step 1: Initialize project structure")
	display.PrintCommand("powershell", "mkdir web-server; cd web-server")
	
	display.PrintTask("Step 2: Create server files")
	display.PrintCommand("powershell", "echo 'console.log(\"Hello World\");' > server.js")
	
	display.PrintTask("Step 3: Install dependencies")
	display.PrintCommand("bash", "npm install express")
	
	display.PrintTask("Step 4: Deploy application")  
	display.PrintCommand("powershell", "npm start")
	
	// Demo: Execution with progress
	display.PrintExecutionStart()
	
	// Simulate execution with progress bar
	progressBar := ui.NewProgressBar(4, "Executing plan")
	
	for i := 1; i <= 4; i++ {
		display.PrintStepStart(i, 4, fmt.Sprintf("Step %d execution", i))
		
		// Simulate work
		time.Sleep(1 * time.Second)
		
		if i == 3 {
			// Simulate a warning
			display.PrintStepSkipped("npm install express", "dependency already exists")
		} else {
			display.PrintStepSuccess(fmt.Sprintf("Step %d command", i))
		}
		
		progressBar.Increment()
	}
	
	progressBar.Complete()
	
	// Demo: Completion
	display.PrintCompletion(3, 4)
	
	// Demo: Various message types
	display.PrintSection("Message Types Demo")
	display.PrintInfo("This is an informational message")
	display.PrintWarning("This is a warning message")
	display.PrintError("This is an error message")
	display.PrintVerbose("This is verbose output (only shown in verbose mode)")
	
	// Demo: Session mutation
	fmt.Println()
	display.PrintSessionMutation("export PATH=$PATH:/usr/local/bin")
	
	fmt.Println()
	display.PrintInfo("UI demonstration complete!")
}