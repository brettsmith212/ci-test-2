package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

// TaskResponse represents a task in API responses
type TaskResponse struct {
	ID        string    `json:"id"`
	Repo      string    `json:"repo"`
	Branch    string    `json:"branch,omitempty"`
	ThreadID  string    `json:"thread_id,omitempty"`
	Prompt    string    `json:"prompt"`
	Status    string    `json:"status"`
	CIRunID   *int64    `json:"ci_run_id,omitempty"`
	Attempts  int       `json:"attempts"`
	Summary   string    `json:"summary,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskListResponse represents the response for listing tasks
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	var statusFilter string
	var limit int
	var offset int
	var outputFormat string
	var watchMode bool
	var repo string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long: `List tasks with optional filtering and pagination.

Examples:
  ampx list                              # List all tasks
  ampx list --status=running             # List running tasks
  ampx list --status=failed --limit=10   # List last 10 failed tasks
  ampx list --repo=github.com/user/repo  # List tasks for specific repo
  ampx list --watch                      # Watch for task changes
  ampx list -o json                      # Output as JSON`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			config, err := cli.LoadConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create client
			client := cli.NewClient(config)

			if watchMode {
				return watchTasks(client, statusFilter, limit, offset, outputFormat, repo)
			}

			return listTasks(client, statusFilter, limit, offset, outputFormat, repo)
		},
	}

	cmd.Flags().StringVarP(&statusFilter, "status", "s", "", "Filter by status (queued, running, retrying, needs_review, success, failed, aborted)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of tasks to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of tasks to skip")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
	cmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for task changes (updates every 5 seconds)")
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "Filter by repository")

	return cmd
}

// listTasks fetches and displays tasks
func listTasks(client *cli.Client, status string, limit, offset int, format, repo string) error {
	// Build query parameters
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if repo != "" {
		params.Set("repo", repo)
	}

	// Build URL path
	path := "/api/v1/tasks"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	// Make API request
	resp, err := client.Get(path)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Parse response
	var listResp TaskListResponse
	if err := client.HandleResponse(resp, &listResp); err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Display results
	switch format {
	case "json":
		return cli.PrintJSON(listResp)
	case "table", "":
		return outputTaskTable(listResp)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// watchTasks continuously watches for task updates
func watchTasks(client *cli.Client, status string, limit, offset int, format, repo string) error {
	fmt.Println("Watching for task updates... (Press Ctrl+C to exit)")
	fmt.Println()

	for {
		if err := listTasks(client, status, limit, offset, format, repo); err != nil {
			return err
		}

		if format == "table" {
			fmt.Println("\n" + strings.Repeat("-", 80))
			fmt.Printf("Updated at: %s\n", time.Now().Format("15:04:05"))
			fmt.Println(strings.Repeat("-", 80))
		}

		time.Sleep(5 * time.Second)
	}
}

// outputTaskTable displays tasks in table format
func outputTaskTable(resp TaskListResponse) error {
	if len(resp.Tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(cli.GetOutput(), 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "ID\tSTATUS\tREPO\tBRANCH\tATTEMPTS\tCREATED\tPROMPT")
	fmt.Fprintln(w, strings.Repeat("-", 8)+"\t"+strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 25)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 8)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 30))

	// Print tasks
	for _, task := range resp.Tasks {
		id := truncateString(task.ID, 8)
		status := formatStatus(task.Status)
		repo := formatRepo(task.Repo)
		branch := truncateString(task.Branch, 15)
		attempts := strconv.Itoa(task.Attempts)
		created := task.CreatedAt.Format("15:04:05")
		prompt := truncateString(task.Prompt, 30)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			id, status, repo, branch, attempts, created, prompt)
	}

	fmt.Fprintf(w, "\nTotal: %d tasks\n", resp.Total)
	return nil
}

// formatStatus adds color/symbols to status
func formatStatus(status string) string {
	switch status {
	case "queued":
		return "⏳ " + status
	case "running":
		return "🔄 " + status
	case "retrying":
		return "🔁 " + status
	case "needs_review":
		return "⚠️  review"
	case "success":
		return "✅ " + status
	case "failed", "error":
		return "❌ " + status
	case "aborted":
		return "🛑 " + status
	default:
		return status
	}
}

// formatRepo extracts repo name from URL
func formatRepo(repoURL string) string {
	// Extract repo name from URL
	parts := strings.Split(repoURL, "/")
	if len(parts) >= 2 {
		repo := parts[len(parts)-1]
		// Remove .git suffix if present
		if strings.HasSuffix(repo, ".git") {
			repo = strings.TrimSuffix(repo, ".git")
		}
		// Include owner/repo format
		if len(parts) >= 3 {
			owner := parts[len(parts)-2]
			return fmt.Sprintf("%s/%s", owner, repo)
		}
		return repo
	}
	return truncateString(repoURL, 25)
}

// truncateString truncates string to specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
