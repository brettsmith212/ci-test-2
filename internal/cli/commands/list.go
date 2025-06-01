package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
	"github.com/brettsmith212/ci-test-2/internal/cli/output"
	"github.com/brettsmith212/ci-test-2/internal/models"
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
		// Convert to models.Task for consistent JSON output
		tasks := make([]models.Task, len(listResp.Tasks))
		for i, t := range listResp.Tasks {
			tasks[i] = models.Task{
				ID:        t.ID,
				Repo:      t.Repo,
				Branch:    t.Branch,
				Prompt:    t.Prompt,
				Status:    models.TaskStatus(t.Status),
				CreatedAt: t.CreatedAt,
				UpdatedAt: t.UpdatedAt,
			}
		}
		formatter := output.NewFormatter(cli.GetOutput(), output.FormatJSON)
		return formatter.FormatTasks(tasks)
	case "wide":
		return outputTaskTableWide(listResp)
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
	// Convert to models.Task for formatter
	tasks := make([]models.Task, len(resp.Tasks))
	for i, t := range resp.Tasks {
		tasks[i] = models.Task{
			ID:        t.ID,
			Repo:      t.Repo,
			Branch:    t.Branch,
			Prompt:    t.Prompt,
			Status:    models.TaskStatus(t.Status),
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}
	}

	formatter := output.NewFormatter(cli.GetOutput(), output.FormatTable)
	if err := formatter.FormatTasks(tasks); err != nil {
		return err
	}

	if resp.Total > len(resp.Tasks) {
		fmt.Fprintf(cli.GetOutput(), "\n%s\n", output.Muted(fmt.Sprintf("Showing %d of %d tasks", len(resp.Tasks), resp.Total)))
	}

	return nil
}

// outputTaskTableWide displays tasks in wide table format
func outputTaskTableWide(resp TaskListResponse) error {
	// Convert to models.Task for formatter
	tasks := make([]models.Task, len(resp.Tasks))
	for i, t := range resp.Tasks {
		tasks[i] = models.Task{
			ID:        t.ID,
			Repo:      t.Repo,
			Branch:    t.Branch,
			Prompt:    t.Prompt,
			Status:    models.TaskStatus(t.Status),
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}
	}

	formatter := output.NewFormatter(cli.GetOutput(), output.FormatWide)
	if err := formatter.FormatTasks(tasks); err != nil {
		return err
	}

	if resp.Total > len(resp.Tasks) {
		fmt.Fprintf(cli.GetOutput(), "\n%s\n", output.Muted(fmt.Sprintf("Showing %d of %d tasks", len(resp.Tasks), resp.Total)))
	}

	return nil
}


