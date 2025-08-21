package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// KeyboardHandler manages keyboard input for interactive features
type KeyboardHandler struct {
	reader *bufio.Reader
}

// NewKeyboardHandler creates a new keyboard handler
func NewKeyboardHandler() *KeyboardHandler {
	return &KeyboardHandler{
		reader: bufio.NewReader(os.Stdin),
	}
}

// WaitForExpandKey waits for user to press a key to expand details
func (kh *KeyboardHandler) WaitForExpandKey(message string) bool {
	if message == "" {
		message = "Press Enter to expand details, or any other key to continue"
	}
	
	Muted.Printf("  %s: ", message)
	
	// Read a single line of input
	input, err := kh.reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	// Empty input (just Enter) means expand
	return strings.TrimSpace(input) == ""
}

// PromptWithOptions presents options and waits for selection
func (kh *KeyboardHandler) PromptWithOptions(message string, options []string, defaultOption int) (int, string) {
	fmt.Printf("\n%s\n", message)
	
	for i, option := range options {
		marker := "  "
		if i == defaultOption {
			marker = "► "
		}
		fmt.Printf("%s%d) %s\n", marker, i+1, option)
	}
	
	fmt.Print("\nSelect option (1-" + fmt.Sprintf("%d", len(options)) + "): ")
	
	input, err := kh.reader.ReadString('\n')
	if err != nil {
		return defaultOption, options[defaultOption]
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultOption, options[defaultOption]
	}
	
	// Try to parse as number
	for i, option := range options {
		if input == fmt.Sprintf("%d", i+1) {
			return i, option
		}
		// Also try partial string matching
		if strings.HasPrefix(strings.ToLower(option), strings.ToLower(input)) {
			return i, option
		}
	}
	
	// Invalid input, return default
	return defaultOption, options[defaultOption]
}

// WaitForAnyKey waits for any key press
func (kh *KeyboardHandler) WaitForAnyKey(message string) {
	if message == "" {
		message = "Press any key to continue"
	}
	
	Muted.Printf("%s...", message)
	kh.reader.ReadString('\n')
}

// PromptYesNo asks a yes/no question
func (kh *KeyboardHandler) PromptYesNo(message string, defaultYes bool) bool {
	suffix := "[Y/n]"
	if !defaultYes {
		suffix = "[y/N]"
	}
	
	fmt.Printf("%s %s: ", message, suffix)
	
	input, err := kh.reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "" {
		return defaultYes
	}
	
	return input == "y" || input == "yes"
}

// Enhanced display with keyboard shortcuts
type InteractiveDisplayEnhanced struct {
	*InteractiveDisplay
	keyboard *KeyboardHandler
	shortcuts map[string]func()
}

// NewInteractiveDisplayEnhanced creates an enhanced interactive display
func NewInteractiveDisplayEnhanced(verbose, debug bool) *InteractiveDisplayEnhanced {
	return &InteractiveDisplayEnhanced{
		InteractiveDisplay: NewInteractiveDisplay(verbose, debug),
		keyboard:          NewKeyboardHandler(),
		shortcuts:         make(map[string]func()),
	}
}

// ShowActionWithShortcuts shows an action with keyboard shortcuts
func (ide *InteractiveDisplayEnhanced) ShowActionWithShortcuts(title, summary string, expandable bool) *InteractiveAction {
	action := ide.ShowAction(title, summary, expandable)
	
	if expandable {
		// Override the summary to include shortcut hint
		originalSummary := action.Summary
		action.Summary = originalSummary
		
		// Print the action with shortcut info
		fmt.Print("\r\033[K") // Clear line
		Secondary.Printf("● %s\n", title)
		if summary != "" {
			Muted.Printf("  ⎿  %s", summary)
			if expandable {
				Muted.Printf(" (Enter=expand, other=skip)")
			}
			fmt.Println()
		}
	}
	
	return action
}

// WaitForExpansionInput waits for expansion input with keyboard shortcuts
func (ide *InteractiveDisplayEnhanced) WaitForExpansionInput(action *InteractiveAction) bool {
	if !action.Expandable {
		return false
	}
	
	return ide.keyboard.WaitForExpandKey("")
}

// ShowContextualHelp shows available keyboard shortcuts
func (ide *InteractiveDisplayEnhanced) ShowContextualHelp() {
	Muted.Println("\nKeyboard shortcuts:")
	Muted.Println("  Enter    - Expand details for current action")
	Muted.Println("  Ctrl+C   - Cancel current operation")
	Muted.Println("  q        - Quit (when prompted)")
	Muted.Println("  h        - Show this help")
}

// Windows-specific keyboard handling (if needed)
var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle = kernel32.NewProc("GetStdHandle")
	procReadConsoleInput = kernel32.NewProc("ReadConsoleInputW")
	procGetNumberOfConsoleInputEvents = kernel32.NewProc("GetNumberOfConsoleInputEvents")
)

const (
	STD_INPUT_HANDLE = ^uintptr(10 - 1) // -10 as uintptr
	KEY_EVENT = 0x0001
)

// InputRecord represents Windows console input record
type InputRecord struct {
	EventType uint16
	KeyEvent  KeyEventRecord
}

// KeyEventRecord represents Windows key event
type KeyEventRecord struct {
	KeyDown         int32
	RepeatCount     uint16
	VirtualKeyCode  uint16
	VirtualScanCode uint16
	UnicodeChar     uint16
	ControlKeyState uint32
}

// IsKeyAvailable checks if a key is available (Windows-specific)
func IsKeyAvailable() bool {
	handle, _, _ := procGetStdHandle.Call(uintptr(STD_INPUT_HANDLE))
	if handle == 0 {
		return false
	}
	
	var numEvents uint32
	ret, _, _ := procGetNumberOfConsoleInputEvents.Call(handle, uintptr(unsafe.Pointer(&numEvents)))
	return ret != 0 && numEvents > 0
}

// GetKeyPress gets a key press without Enter (Windows-specific, simplified)
func GetKeyPress() (rune, error) {
	// For simplicity, we'll use the standard reader
	// In a full implementation, this would use Windows console APIs
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return '\n', nil
	}
	
	return rune(input[0]), nil
}

// SmartPrompt provides intelligent prompting based on context
func (ide *InteractiveDisplayEnhanced) SmartPrompt(context string, options []string) string {
	if len(options) == 0 {
		return ""
	}
	
	// For single option, auto-select
	if len(options) == 1 {
		Muted.Printf("Auto-selecting: %s\n", options[0])
		return options[0]
	}
	
	// For two options where one is clearly default (e.g., yes/no)
	if len(options) == 2 {
		if strings.ToLower(options[0]) == "yes" || strings.ToLower(options[0]) == "y" {
			return ide.promptYesNoSmart(context, true)
		} else if strings.ToLower(options[1]) == "yes" || strings.ToLower(options[1]) == "y" {
			return ide.promptYesNoSmart(context, false)
		}
	}
	
	// Multi-option selection
	index, selected := ide.keyboard.PromptWithOptions(context, options, 0)
	_ = index // Use index for future enhancements
	return selected
}

// promptYesNoSmart provides smart yes/no prompting
func (ide *InteractiveDisplayEnhanced) promptYesNoSmart(message string, defaultYes bool) string {
	if ide.keyboard.PromptYesNo(message, defaultYes) {
		return "yes"
	}
	return "no"
}

// SetupGlobalShortcuts sets up global keyboard shortcuts
func (ide *InteractiveDisplayEnhanced) SetupGlobalShortcuts() {
	ide.shortcuts["h"] = ide.ShowContextualHelp
	ide.shortcuts["help"] = ide.ShowContextualHelp
	ide.shortcuts["?"] = ide.ShowContextualHelp
}

// ProcessShortcut processes a keyboard shortcut
func (ide *InteractiveDisplayEnhanced) ProcessShortcut(key string) bool {
	if handler, exists := ide.shortcuts[strings.ToLower(key)]; exists {
		handler()
		return true
	}
	return false
}