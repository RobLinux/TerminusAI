package common

import (
	"runtime"
	"testing"
)

func TestCopyToClipboard(t *testing.T) {
	testText := "test-clipboard-content"

	err := CopyToClipboard(testText)
	if err != nil {
		// Only fail if we're on a supported platform
		switch runtime.GOOS {
		case "windows", "darwin":
			t.Fatalf("Expected clipboard copy to work on %s, got error: %v", runtime.GOOS, err)
		case "linux":
			// On Linux, it's expected to fail in CI/testing environments
			t.Logf("Clipboard copy failed on Linux (expected in testing): %v", err)
		default:
			t.Logf("Clipboard copy failed on unsupported platform %s: %v", runtime.GOOS, err)
		}
	} else {
		t.Logf("Successfully copied '%s' to clipboard", testText)
	}
}
