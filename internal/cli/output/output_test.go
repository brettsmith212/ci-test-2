package output

import (
	"os"
	"strings"
	"testing"
)

func TestIsColorEnabled(t *testing.T) {
	// Save original env vars
	originalNoColor := os.Getenv("NO_COLOR")
	originalAmpxColor := os.Getenv("AMPX_COLOR")
	originalTerm := os.Getenv("TERM")
	
	defer func() {
		os.Setenv("NO_COLOR", originalNoColor)
		os.Setenv("AMPX_COLOR", originalAmpxColor)
		os.Setenv("TERM", originalTerm)
	}()

	tests := []struct {
		name     string
		noColor  string
		ampxColor string
		term     string
		expected bool
	}{
		{
			name:     "default with color terminal",
			noColor:  "",
			ampxColor: "",
			term:     "xterm-256color",
			expected: true,
		},
		{
			name:     "NO_COLOR set",
			noColor:  "1",
			ampxColor: "",
			term:     "xterm-256color",
			expected: false,
		},
		{
			name:     "AMPX_COLOR force enabled",
			noColor:  "",
			ampxColor: "1",
			term:     "dumb",
			expected: true,
		},
		{
			name:     "dumb terminal",
			noColor:  "",
			ampxColor: "",
			term:     "dumb",
			expected: false,
		},
		{
			name:     "no terminal",
			noColor:  "",
			ampxColor: "",
			term:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("NO_COLOR", tt.noColor)
			os.Setenv("AMPX_COLOR", tt.ampxColor)
			os.Setenv("TERM", tt.term)

			result := IsColorEnabled()
			if result != tt.expected {
				t.Errorf("Expected IsColorEnabled() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestColorize(t *testing.T) {
	// Save original env vars
	originalNoColor := os.Getenv("NO_COLOR")
	originalAmpxColor := os.Getenv("AMPX_COLOR")
	originalTerm := os.Getenv("TERM")
	
	defer func() {
		os.Setenv("NO_COLOR", originalNoColor)
		os.Setenv("AMPX_COLOR", originalAmpxColor)
		os.Setenv("TERM", originalTerm)
	}()

	// Test with colors enabled
	os.Setenv("AMPX_COLOR", "1")
	os.Setenv("TERM", "xterm-256color")
	os.Unsetenv("NO_COLOR")
	
	result := Colorize("test", Red)
	if !strings.Contains(result, "test") {
		t.Errorf("Expected result to contain 'test', got %s", result)
	}

	// Test with colors disabled
	os.Setenv("NO_COLOR", "1")
	os.Unsetenv("AMPX_COLOR")
	
	result = Colorize("test", Red)
	if result != "test" {
		t.Errorf("Expected plain text with colors disabled, got %s", result)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected Color
	}{
		{"queued", Yellow},
		{"running", Blue},
		{"completed", BrightGreen},
		{"failed", Red},
		{"aborted", BrightRed},
		{"unknown", Color("")}, // Should return original string without color
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := Status(tt.status)
			
			if tt.expected != "" {
				// Should contain the status text
				if !strings.Contains(result, tt.status) {
					t.Errorf("Expected result to contain status '%s', got %s", tt.status, result)
				}
			} else {
				// Unknown status should return as-is
				if result != tt.status {
					t.Errorf("Expected unknown status to return as-is, got %s", result)
				}
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate with ellipsis",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "very short max length",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "max length of 3",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSemanticFormatters(t *testing.T) {
	// Test that semantic formatters return non-empty strings
	testText := "test message"
	
	formatters := map[string]func(string) string{
		"Success":    Success,
		"Error":      Error,
		"Warning":    Warning,
		"Info":       Info,
		"Primary":    Primary,
		"Secondary":  Secondary,
		"Muted":      Muted,
		"Header":     Header,
		"Subheader":  Subheader,
		"Code":       Code,
		"URL":        URL,
		"ID":         ID,
		"Timestamp":  Timestamp,
		"Branch":     Branch,
		"Repository": Repository,
	}

	for name, formatter := range formatters {
		t.Run(name, func(t *testing.T) {
			result := formatter(testText)
			if result == "" {
				t.Errorf("Formatter %s returned empty string", name)
			}
			// Should contain the original text
			if !strings.Contains(result, testText) {
				t.Errorf("Formatter %s result should contain original text '%s', got '%s'", name, testText, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkColorize(b *testing.B) {
	text := "benchmark test text"
	color := BrightBlue
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Colorize(text, color)
	}
}

func BenchmarkTruncateString(b *testing.B) {
	text := "This is a long string that will need to be truncated for display purposes"
	maxLen := 50
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TruncateString(text, maxLen)
	}
}

func BenchmarkStatus(b *testing.B) {
	statuses := []string{"queued", "running", "completed", "failed", "aborted"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		Status(status)
	}
}
