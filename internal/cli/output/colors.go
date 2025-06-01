package output

import (
	"fmt"
	"os"
	"strings"
)

type Color string

const (
	Reset   Color = "\033[0m"
	Bold    Color = "\033[1m"
	Dim     Color = "\033[2m"
	Italic  Color = "\033[3m"
	Underline Color = "\033[4m"

	// Standard colors
	Black   Color = "\033[30m"
	Red     Color = "\033[31m"
	Green   Color = "\033[32m"
	Yellow  Color = "\033[33m"
	Blue    Color = "\033[34m"
	Magenta Color = "\033[35m"
	Cyan    Color = "\033[36m"
	White   Color = "\033[37m"

	// Bright colors
	BrightBlack   Color = "\033[90m"
	BrightRed     Color = "\033[91m"
	BrightGreen   Color = "\033[92m"
	BrightYellow  Color = "\033[93m"
	BrightBlue    Color = "\033[94m"
	BrightMagenta Color = "\033[95m"
	BrightCyan    Color = "\033[96m"
	BrightWhite   Color = "\033[97m"

	// Background colors
	BgBlack   Color = "\033[40m"
	BgRed     Color = "\033[41m"
	BgGreen   Color = "\033[42m"
	BgYellow  Color = "\033[43m"
	BgBlue    Color = "\033[44m"
	BgMagenta Color = "\033[45m"
	BgCyan    Color = "\033[46m"
	BgWhite   Color = "\033[47m"
)

// ColorScheme defines our application's color palette
type ColorScheme struct {
	Primary     Color
	Secondary   Color
	Success     Color
	Warning     Color
	Error       Color
	Info        Color
	Muted       Color
	Accent      Color
	Background  Color
}

var (
	DefaultScheme = ColorScheme{
		Primary:    BrightBlue,
		Secondary:  Cyan,
		Success:    BrightGreen,
		Warning:    BrightYellow,
		Error:      BrightRed,
		Info:       BrightCyan,
		Muted:      BrightBlack,
		Accent:     BrightMagenta,
		Background: Reset,
	}

	// Status-specific colors
	StatusColors = map[string]Color{
		"queued":     Yellow,
		"running":    Blue,
		"completed":  BrightGreen,
		"failed":     Red,
		"aborted":    BrightRed,
		"continued":  Cyan,
	}

	// Priority colors
	PriorityColors = map[string]Color{
		"low":    BrightBlack,
		"medium": Yellow,
		"high":   BrightRed,
	}
)

// IsColorEnabled checks if color output should be enabled
func IsColorEnabled() bool {
	// Check if explicitly disabled
	if val := os.Getenv("NO_COLOR"); val != "" {
		return false
	}
	if val := os.Getenv("AMPX_NO_COLOR"); val != "" {
		return false
	}

	// Check if explicitly enabled
	if val := os.Getenv("FORCE_COLOR"); val != "" {
		return true
	}
	if val := os.Getenv("AMPX_COLOR"); val != "" {
		return true
	}

	// Check if terminal supports color
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	// Default to true for most terminals
	return true
}

// Colorize applies color to text if colors are enabled
func Colorize(text string, color Color) string {
	if !IsColorEnabled() {
		return text
	}
	return string(color) + text + string(Reset)
}

// ColorizeWithReset applies color and ensures reset
func ColorizeWithReset(text string, color Color) string {
	if !IsColorEnabled() {
		return text
	}
	return string(color) + text + string(Reset)
}

// Sprintf with color support
func Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// Status returns a colored status string
func Status(status string) string {
	if color, exists := StatusColors[strings.ToLower(status)]; exists {
		return Colorize(status, color)
	}
	return status
}

// Priority returns a colored priority string
func Priority(priority string) string {
	if color, exists := PriorityColors[strings.ToLower(priority)]; exists {
		return Colorize(priority, color)
	}
	return priority
}

// Success formats success messages
func Success(text string) string {
	return Colorize(text, DefaultScheme.Success)
}

// Error formats error messages
func Error(text string) string {
	return Colorize(text, DefaultScheme.Error)
}

// Warning formats warning messages
func Warning(text string) string {
	return Colorize(text, DefaultScheme.Warning)
}

// Info formats info messages
func Info(text string) string {
	return Colorize(text, DefaultScheme.Info)
}

// Muted formats muted/secondary text
func Muted(text string) string {
	return Colorize(text, DefaultScheme.Muted)
}

// Primary formats primary text
func Primary(text string) string {
	return Colorize(text, DefaultScheme.Primary)
}

// Secondary formats secondary text
func Secondary(text string) string {
	return Colorize(text, DefaultScheme.Secondary)
}

// Bold formats bold text
func BoldText(text string) string {
	return Colorize(text, Bold)
}

// Underline formats underlined text
func UnderlineText(text string) string {
	return Colorize(text, Underline)
}

// Header formats header text (bold + primary color)
func Header(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return string(Bold) + string(DefaultScheme.Primary) + text + string(Reset)
}

// Subheader formats subheader text (secondary color)
func Subheader(text string) string {
	return Secondary(text)
}

// Code formats code/monospace text
func Code(text string) string {
	return Colorize(text, BrightWhite)
}

// URL formats URLs
func URL(text string) string {
	return Colorize(text, BrightCyan)
}

// ID formats IDs (like task IDs)
func ID(text string) string {
	return Colorize(text, BrightMagenta)
}

// Timestamp formats timestamps
func Timestamp(text string) string {
	return Muted(text)
}

// Branch formats git branch names
func Branch(text string) string {
	return Colorize(text, BrightGreen)
}

// Repository formats repository names
func Repository(text string) string {
	return Colorize(text, BrightCyan)
}
