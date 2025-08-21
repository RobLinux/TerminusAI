package common

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetStringWithDefault returns the string value or a default if empty
func GetStringWithDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}