package agent

import (
	"errors"
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"empty string", "", 10, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"long string", "hello world", 5, "hello"},
		{"zero length", "hello", 0, ""},
		{"unicode string", "héllo wørld", 5, "héllo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a smaller", 1, 2, 1},
		{"b smaller", 5, 3, 3},
		{"equal", 4, 4, 4},
		{"negative numbers", -5, -2, -5},
		{"zero", 0, 5, 0},
		{"negative and positive", -3, 7, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"overloaded error", errors.New("API is overloaded"), true},
		{"timeout error", errors.New("Request timeout"), true},
		{"rate limit error", errors.New("Rate limit exceeded"), true},
		{"502 error", errors.New("502 Bad Gateway"), true},
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"504 error", errors.New("504 Gateway Timeout"), true},
		{"case insensitive overloaded", errors.New("API is OVERLOADED"), true},
		{"case insensitive timeout", errors.New("REQUEST TIMEOUT"), true},
		{"non-retryable error", errors.New("Invalid API key"), false},
		{"400 error", errors.New("400 Bad Request"), false},
		{"404 error", errors.New("404 Not Found"), false},
		{"authentication error", errors.New("Authentication failed"), false},
		{"empty error message", errors.New(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				errorMsg := "nil"
				if tt.err != nil {
					errorMsg = tt.err.Error()
				}
				t.Errorf("isRetryableError(%q) = %v, want %v", errorMsg, result, tt.expected)
			}
		})
	}
}
