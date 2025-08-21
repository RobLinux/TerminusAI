package ui

import (
	"fmt"
	"sync"
	"time"
)

// StatusIndicator manages real-time status updates
type StatusIndicator struct {
	mu          sync.RWMutex
	isActive    bool
	currentTask string
	startTime   time.Time
	statusChan  chan string
	doneChan    chan struct{}
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator() *StatusIndicator {
	return &StatusIndicator{
		statusChan: make(chan string, 10),
		doneChan:   make(chan struct{}),
	}
}

// Start begins the status indicator
func (s *StatusIndicator) Start(initialTask string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isActive {
		return
	}
	
	s.isActive = true
	s.currentTask = initialTask
	s.startTime = time.Now()
	
	go s.run()
}

// UpdateTask updates the current task being displayed
func (s *StatusIndicator) UpdateTask(task string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isActive {
		return
	}
	
	s.currentTask = task
	select {
	case s.statusChan <- task:
	default:
		// Channel full, skip this update
	}
}

// Stop halts the status indicator
func (s *StatusIndicator) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isActive {
		return
	}
	
	s.isActive = false
	close(s.doneChan)
	
	// Clear the status line
	fmt.Print("\r\033[K")
}

// run handles the main status display loop
func (s *StatusIndicator) run() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIndex := 0
	
	for {
		select {
		case <-s.doneChan:
			return
		case newTask := <-s.statusChan:
			s.mu.Lock()
			s.currentTask = newTask
			s.mu.Unlock()
		case <-ticker.C:
			s.mu.RLock()
			if !s.isActive {
				s.mu.RUnlock()
				return
			}
			
			elapsed := time.Since(s.startTime).Round(time.Second)
			frame := spinnerFrames[frameIndex%len(spinnerFrames)]
			frameIndex++
			
			Processing.Printf("\r%s %s [%v]", frame, s.currentTask, elapsed)
			s.mu.RUnlock()
		}
	}
}

// CommandExecutor represents a command being executed with real-time feedback
type CommandExecutor struct {
	command     string
	status      *StatusIndicator
	startTime   time.Time
	isRunning   bool
	mu          sync.RWMutex
}

// NewCommandExecutor creates a new command executor with status tracking
func NewCommandExecutor(command string) *CommandExecutor {
	return &CommandExecutor{
		command: command,
		status:  NewStatusIndicator(),
	}
}

// Start begins command execution display
func (c *CommandExecutor) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.isRunning {
		return
	}
	
	c.isRunning = true
	c.startTime = time.Now()
	
	c.status.Start(fmt.Sprintf("Executing: %s", c.command))
}

// UpdateProgress updates the execution progress
func (c *CommandExecutor) UpdateProgress(message string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.isRunning {
		return
	}
	
	c.status.UpdateTask(fmt.Sprintf("%s: %s", c.command, message))
}

// Complete marks the command as completed
func (c *CommandExecutor) Complete(success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.isRunning {
		return
	}
	
	c.isRunning = false
	c.status.Stop()
	
	elapsed := time.Since(c.startTime).Round(time.Millisecond)
	
	if success {
		Success.Printf("✓ %s [%v]\n", c.command, elapsed)
	} else {
		Error.Printf("✗ %s [%v]\n", c.command, elapsed)
	}
}

// IsRunning checks if the command is currently executing
func (c *CommandExecutor) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// LiveLogDisplay manages streaming log output with colors and formatting
type LiveLogDisplay struct {
	mu       sync.RWMutex
	active   bool
	maxLines int
	lines    []string
}

// NewLiveLogDisplay creates a new live log display
func NewLiveLogDisplay(maxLines int) *LiveLogDisplay {
	return &LiveLogDisplay{
		maxLines: maxLines,
		lines:    make([]string, 0, maxLines),
	}
}

// AddLine adds a new line to the live display
func (l *LiveLogDisplay) AddLine(line string, level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Format line based on level
	var formattedLine string
	switch level {
	case "error":
		formattedLine = Error.Sprintf("ERROR: %s", line)
	case "warning":  
		formattedLine = Warning.Sprintf("WARN:  %s", line)
	case "info":
		formattedLine = Secondary.Sprintf("INFO:  %s", line)
	case "debug":
		formattedLine = Muted.Sprintf("DEBUG: %s", line)
	default:
		formattedLine = line
	}
	
	// Add to lines buffer
	l.lines = append(l.lines, formattedLine)
	
	// Keep only the most recent lines
	if len(l.lines) > l.maxLines {
		l.lines = l.lines[len(l.lines)-l.maxLines:]
	}
	
	// Print the new line
	fmt.Println(formattedLine)
}

// Clear clears the log display
func (l *LiveLogDisplay) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.lines = l.lines[:0]
	
	// Clear screen (optional)
	fmt.Print("\033[H\033[2J")
}

// GetLines returns a copy of all current lines
func (l *LiveLogDisplay) GetLines() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	result := make([]string, len(l.lines))
	copy(result, l.lines)
	return result
}

// TaskTimer tracks timing for individual tasks
type TaskTimer struct {
	name      string
	startTime time.Time
	isRunning bool
	mu        sync.RWMutex
}

// NewTaskTimer creates a new task timer
func NewTaskTimer(name string) *TaskTimer {
	return &TaskTimer{
		name: name,
	}
}

// Start begins timing
func (t *TaskTimer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.startTime = time.Now()
	t.isRunning = true
	
	Muted.Printf("⏱ Started: %s\n", t.name)
}

// Stop ends timing and displays result
func (t *TaskTimer) Stop() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if !t.isRunning {
		return 0
	}
	
	elapsed := time.Since(t.startTime)
	t.isRunning = false
	
	Muted.Printf("⏱ Completed: %s [%v]\n", t.name, elapsed.Round(time.Millisecond))
	return elapsed
}

// Elapsed returns the current elapsed time
func (t *TaskTimer) Elapsed() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if !t.isRunning {
		return 0
	}
	
	return time.Since(t.startTime)
}