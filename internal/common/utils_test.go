package common

import (
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
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
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
			result := Min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestGetStringWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue string
		expected     string
	}{
		{"non-empty value", "hello", "default", "hello"},
		{"empty value", "", "default", "default"},
		{"empty value and default", "", "", ""},
		{"whitespace value", "   ", "default", "   "},
		{"special characters", "test@#$", "default", "test@#$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringWithDefault(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetStringWithDefault(%q, %q) = %q, want %q", tt.value, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
