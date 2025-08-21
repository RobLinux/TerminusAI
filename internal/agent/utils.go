package agent

import (
	"strings"
)

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isRetryableError checks if an error is worth retrying (API overload, timeout, etc.)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "overloaded") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504")
}
