package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
	"github.com/brettsmith212/ci-test-2/internal/cli/output"
	"github.com/brettsmith212/ci-test-2/internal/models"
)

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
	var followFlag bool
	var tailLines int
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "Show logs for a task",
		Long: `Show logs and detailed information for a specific task.

This command displays the task details including status, prompt, repository,
and any available logs or summary information.

Examples:
  ampx logs abc123                    # Show logs for task abc123
  ampx logs abc123 --follow           # Follow logs in real-time
  ampx logs abc123 --tail=50          # Show last 50 lines
  ampx logs abc123 -o json            # Output as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Load configuration
			config, err := cli.LoadConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create client
			client := cli.NewClient(config)

			if followFlag {
				return followTaskLogs(client, taskID, outputFormat)
			}

			return showTaskLogs(client, taskID, tailLines, outputFormat)
		},
	}

	cmd.Flags().BoolVarP(&followFlag, "follow", "f", false, "Follow logs in real-time")
	cmd.Flags().IntVarP(&tailLines, "tail", "t", 100, "Number of lines to show from the end")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// showTaskLogs displays logs for a task
func showTaskLogs(client *cli.Client, taskID string, tailLines int, format string) error {
	// Get task details
	resp, err := client.Get(fmt.Sprintf("/api/v1/tasks/%s", taskID))
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	var task TaskResponse
	if err := client.HandleResponse(resp, &task); err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Convert to models.Task for formatter
	modelTask := models.Task{
		ID:        task.ID,
		Repo:      task.Repo,
		Branch:    task.Branch,
		ThreadID:  task.ThreadID,
		Prompt:    task.Prompt,
		Status:    models.TaskStatus(task.Status),
		CIRunID:   task.CIRunID,
		Attempts:  task.Attempts,
		Summary:   task.Summary,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}

	// Display based on format
	switch format {
	case "json":
		formatter := output.NewFormatter(cli.GetOutput(), output.FormatJSON)
		return formatter.FormatTask(modelTask)
	case "table", "":
		formatter := output.NewFormatter(cli.GetOutput(), output.FormatTable)
		return formatter.FormatTask(modelTask)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// followTaskLogs follows task logs in real-time
func followTaskLogs(client *cli.Client, taskID string, format string) error {
	fmt.Printf("Following logs for task %s... (Press Ctrl+C to exit)\n", taskID)
	fmt.Println()

	var lastStatus string
	var lastUpdate time.Time

	for {
		// Get current task status
		resp, err := client.Get(fmt.Sprintf("/api/v1/tasks/%s", taskID))
		if err != nil {
			fmt.Printf("Error fetching task: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var task TaskResponse
		if err := client.HandleResponse(resp, &task); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Check if task has been updated
		if task.Status != lastStatus || task.UpdatedAt.After(lastUpdate) {
			if format == "json" {
				cli.PrintJSON(task)
			} else {
				outputTaskUpdate(task, lastStatus)
			}
			lastStatus = task.Status
			lastUpdate = task.UpdatedAt
		}

		// If task is in terminal state, stop following
		if isTerminalStatus(task.Status) {
			fmt.Printf("\n✓ Task completed with status: %s\n", output.Status(task.Status))
			break
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

// outputTaskLogs displays detailed task information
func outputTaskLogs(task TaskResponse) error {
	fmt.Printf("Task Details: %s\n", task.ID)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Status:      %s\n", output.Status(task.Status))
	fmt.Printf("Repository:  %s\n", task.Repo)
	if task.Branch != "" {
		fmt.Printf("Branch:      %s\n", task.Branch)
	}
	if task.ThreadID != "" {
		fmt.Printf("Thread ID:   %s\n", task.ThreadID)
	}
	fmt.Printf("Attempts:    %d\n", task.Attempts)
	fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
	
	if task.CIRunID != nil {
		fmt.Printf("CI Run ID:   %d\n", *task.CIRunID)
	}

	fmt.Println()
	fmt.Println("Prompt:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(task.Prompt)

	if task.Summary != "" {
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(task.Summary)
	}

	// Show status-specific information
	fmt.Println()
	fmt.Println("Status Information:")
	fmt.Println(strings.Repeat("-", 50))
	
	switch task.Status {
	case "queued":
		fmt.Println("Task is waiting to be processed by a worker.")
	case "running":
		fmt.Println("Task is currently being processed by Amp.")
		if task.CIRunID != nil {
			fmt.Printf("Monitor CI run at: (CI Run ID: %d)\n", *task.CIRunID)
		}
	case "retrying":
		fmt.Printf("Task failed and is being retried (attempt %d).\n", task.Attempts)
	case "needs_review":
		fmt.Println("Task requires manual review before proceeding.")
		fmt.Println("Use 'ampx continue " + task.ID + "' to resume with modifications.")
	case "success":
		fmt.Println("Task completed successfully!")
		if task.CIRunID != nil {
			fmt.Printf("CI run passed (Run ID: %d)\n", *task.CIRunID)
		}
	case "failed", "error":
		fmt.Printf("Task failed after %d attempts.\n", task.Attempts)
		fmt.Println("Use 'ampx continue " + task.ID + "' to retry with modifications.")
	case "aborted":
		fmt.Println("Task was manually aborted.")
	}

	// Show available actions
	fmt.Println()
	fmt.Println("Available Actions:")
	fmt.Println(strings.Repeat("-", 50))
	
	if canContinue(task.Status) {
		fmt.Println("• ampx continue " + task.ID + " \"modified prompt\" - Resume with changes")
	}
	if canAbort(task.Status) {
		fmt.Println("• ampx abort " + task.ID + " - Abort this task")
	}
	if task.Status == "success" {
		fmt.Println("• ampx merge " + task.ID + " - Merge the changes")
	}

	return nil
}

// outputTaskUpdate displays a task status update
func outputTaskUpdate(task TaskResponse, lastStatus string) {
	timestamp := time.Now().Format("15:04:05")
	
	if lastStatus == "" {
		fmt.Printf("[%s] Task %s: %s\n", timestamp, task.ID[:8], output.Status(task.Status))
	} else {
		fmt.Printf("[%s] Task %s: %s → %s\n", timestamp, task.ID[:8], output.Status(lastStatus), output.Status(task.Status))
	}

	if task.Summary != "" {
		fmt.Printf("          Summary: %s\n", task.Summary)
	}
	
	if task.CIRunID != nil {
		fmt.Printf("          CI Run: %d\n", *task.CIRunID)
	}
	
	fmt.Println()
}

// isTerminalStatus checks if a status is terminal
func isTerminalStatus(status string) bool {
	terminalStates := []string{"success", "failed", "error", "aborted"}
	for _, terminal := range terminalStates {
		if status == terminal {
			return true
		}
	}
	return false
}

// canContinue checks if a task can be continued
func canContinue(status string) bool {
	continuableStates := []string{"failed", "error", "retrying", "needs_review"}
	for _, continuable := range continuableStates {
		if status == continuable {
			return true
		}
	}
	return false
}

// canAbort checks if a task can be aborted
func canAbort(status string) bool {
	abortableStates := []string{"queued", "running", "retrying", "needs_review"}
	for _, abortable := range abortableStates {
		if status == abortable {
			return true
		}
	}
	return false
}
