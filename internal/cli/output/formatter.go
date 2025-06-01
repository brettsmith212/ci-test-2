package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/models"
)

type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatWide  OutputFormat = "wide"
)

type Formatter struct {
	writer io.Writer
	format OutputFormat
	colors bool
}

func NewFormatter(writer io.Writer, format OutputFormat) *Formatter {
	return &Formatter{
		writer: writer,
		format: format,
		colors: IsColorEnabled(),
	}
}

func NewDefaultFormatter() *Formatter {
	return NewFormatter(os.Stdout, FormatTable)
}

// FormatTasks formats a list of tasks according to the output format
func (f *Formatter) FormatTasks(tasks []models.Task) error {
	switch f.format {
	case FormatJSON:
		return f.formatTasksJSON(tasks)
	case FormatWide:
		return f.formatTasksWide(tasks)
	default:
		return f.formatTasksTable(tasks)
	}
}

// FormatTask formats a single task with detailed information
func (f *Formatter) FormatTask(task models.Task) error {
	switch f.format {
	case FormatJSON:
		return f.formatTaskJSON(task)
	default:
		return f.formatTaskDetailed(task)
	}
}

func (f *Formatter) formatTasksTable(tasks []models.Task) error {
	if len(tasks) == 0 {
		fmt.Fprintln(f.writer, Muted("No tasks found"))
		return nil
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	header := "ID\tSTATUS\tREPOSITORY\tPROMPT\tCREATED"
	if f.colors {
		header = Header("ID") + "\t" + Header("STATUS") + "\t" + Header("REPOSITORY") + "\t" + Header("PROMPT") + "\t" + Header("CREATED")
	}
	fmt.Fprintln(w, header)

	// Tasks
	for _, task := range tasks {
		id := f.formatID(task.ID)
		status := f.formatStatus(task.Status)
		repo := f.formatRepository(task.Repo)
		prompt := f.formatPrompt(task.Prompt, 50)
		created := f.formatTime(task.CreatedAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, status, repo, prompt, created)
	}

	return nil
}

func (f *Formatter) formatTasksWide(tasks []models.Task) error {
	if len(tasks) == 0 {
		fmt.Fprintln(f.writer, Muted("No tasks found"))
		return nil
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	header := "ID\tSTATUS\tREPOSITORY\tBRANCH\tPROMPT\tCREATED\tUPDATED"
	if f.colors {
		header = Header("ID") + "\t" + Header("STATUS") + "\t" + Header("REPOSITORY") + "\t" + Header("BRANCH") + "\t" + Header("PROMPT") + "\t" + Header("CREATED") + "\t" + Header("UPDATED")
	}
	fmt.Fprintln(w, header)

	// Tasks
	for _, task := range tasks {
		id := f.formatID(task.ID)
		status := f.formatStatus(task.Status)
		repo := f.formatRepository(task.Repo)
		branch := f.formatBranch(task.Branch)
		prompt := f.formatPrompt(task.Prompt, 60)
		created := f.formatTime(task.CreatedAt)
		updated := f.formatTime(task.UpdatedAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", id, status, repo, branch, prompt, created, updated)
	}

	return nil
}

func (f *Formatter) formatTasksJSON(tasks []models.Task) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tasks)
}

func (f *Formatter) formatTaskDetailed(task models.Task) error {
	fmt.Fprintln(f.writer, Header(fmt.Sprintf("Task %s", f.formatID(task.ID))))
	fmt.Fprintln(f.writer)

	// Basic information
	fmt.Fprintf(f.writer, "%-12s %s\n", Primary("Status:"), f.formatStatus(task.Status))
	fmt.Fprintf(f.writer, "%-12s %s\n", Primary("Repository:"), f.formatRepository(task.Repo))
	if task.Branch != "" {
		fmt.Fprintf(f.writer, "%-12s %s\n", Primary("Branch:"), f.formatBranch(task.Branch))
	}
	if task.ThreadID != "" {
		fmt.Fprintf(f.writer, "%-12s %s\n", Primary("Thread ID:"), task.ThreadID)
	}
	if task.CIRunID != nil {
		fmt.Fprintf(f.writer, "%-12s %d\n", Primary("CI Run ID:"), *task.CIRunID)
	}
	fmt.Fprintf(f.writer, "%-12s %d\n", Primary("Attempts:"), task.Attempts)
	fmt.Fprintf(f.writer, "%-12s %s\n", Primary("Created:"), f.formatTime(task.CreatedAt))
	fmt.Fprintf(f.writer, "%-12s %s\n", Primary("Updated:"), f.formatTime(task.UpdatedAt))

	fmt.Fprintln(f.writer)

	// Prompt
	fmt.Fprintln(f.writer, Primary("Prompt:"))
	fmt.Fprintln(f.writer, f.formatLongText(task.Prompt))

	// Summary if available
	if task.Summary != "" {
		fmt.Fprintln(f.writer)
		fmt.Fprintln(f.writer, Primary("Summary:"))
		fmt.Fprintln(f.writer, f.formatLongText(task.Summary))
	}

	return nil
}

func (f *Formatter) formatTaskJSON(task models.Task) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(task)
}

// Helper functions for formatting specific fields

func (f *Formatter) formatID(id string) string {
	if f.colors {
		return ID(id)
	}
	return id
}

func (f *Formatter) formatStatus(status models.TaskStatus) string {
	statusStr := string(status)
	if f.colors {
		return Status(statusStr)
	}
	return statusStr
}

func (f *Formatter) formatRepository(repo string) string {
	// Extract just the repo name from URL for display
	parts := strings.Split(repo, "/")
	if len(parts) >= 2 {
		repoName := parts[len(parts)-1]
		repoName = strings.TrimSuffix(repoName, ".git")
		ownerRepo := parts[len(parts)-2] + "/" + repoName
		
		if f.colors {
			return Repository(ownerRepo)
		}
		return ownerRepo
	}
	
	if f.colors {
		return Repository(repo)
	}
	return repo
}

func (f *Formatter) formatBranch(branch string) string {
	if branch == "" {
		return Muted("-")
	}
	if f.colors {
		return Branch(branch)
	}
	return branch
}

func (f *Formatter) formatPrompt(prompt string, maxLen int) string {
	if len(prompt) <= maxLen {
		return prompt
	}
	
	truncated := prompt[:maxLen-3] + "..."
	if f.colors {
		return Muted(truncated)
	}
	return truncated
}

func (f *Formatter) formatTime(t time.Time) string {
	if t.IsZero() {
		return Muted("-")
	}
	
	now := time.Now()
	diff := now.Sub(t)
	
	var timeStr string
	switch {
	case diff < time.Minute:
		timeStr = "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		timeStr = fmt.Sprintf("%dm ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		timeStr = fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		timeStr = fmt.Sprintf("%dd ago", days)
	default:
		timeStr = t.Format("Jan 2, 2006")
	}
	
	if f.colors {
		return Timestamp(timeStr)
	}
	return timeStr
}

func (f *Formatter) formatLongText(text string) string {
	// Add indentation to long text
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

// Utility functions for common formatting patterns

func FormatTasksTable(tasks []models.Task, writer io.Writer) error {
	formatter := NewFormatter(writer, FormatTable)
	return formatter.FormatTasks(tasks)
}

func FormatTasksJSON(tasks []models.Task, writer io.Writer) error {
	formatter := NewFormatter(writer, FormatJSON)
	return formatter.FormatTasks(tasks)
}

func FormatTaskDetailed(task models.Task, writer io.Writer) error {
	formatter := NewFormatter(writer, FormatTable)
	return formatter.FormatTask(task)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Println(Success("✓ " + message))
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Println(Error("✗ " + message))
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Println(Warning("⚠ " + message))
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Println(Info("ℹ " + message))
}

// PrintHeader prints a header with separator
func PrintHeader(title string) {
	fmt.Println(Header(title))
	if IsColorEnabled() {
		fmt.Println(Muted(strings.Repeat("─", len(title))))
	} else {
		fmt.Println(strings.Repeat("-", len(title)))
	}
}

// PrintSubheader prints a subheader
func PrintSubheader(title string) {
	fmt.Println(Subheader(title))
}

// PrintSeparator prints a visual separator
func PrintSeparator() {
	if IsColorEnabled() {
		fmt.Println(Muted("────────────────────────────────────────"))
	} else {
		fmt.Println("----------------------------------------")
	}
}

// PrintKeyValue prints a key-value pair with consistent formatting
func PrintKeyValue(key, value string) {
	fmt.Printf("%-12s %s\n", Primary(key+":"), value)
}

// TruncateString truncates string to specified length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
