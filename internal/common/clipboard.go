package common

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies the given text to the system clipboard
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use PowerShell to set clipboard on Windows
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Set-Clipboard -Value '%s'", text))
	case "darwin":
		// Use pbcopy on macOS
		cmd = exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
			cmd.Stdin = strings.NewReader(text)
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
			cmd.Stdin = strings.NewReader(text)
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	return cmd.Run()
}
