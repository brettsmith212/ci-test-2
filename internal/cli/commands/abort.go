package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
	"github.com/brettsmith212/ci-test-2/internal/cli/output"
)

// NewAbortCommand creates the abort command
func NewAbortCommand() *cobra.Command {
	var forceFlag bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "abort <task-id>",
		Short: "Abort a running or queued task",
		Long: `Abort a running or queued task.

This command will stop a task that is currently queued, running, or retrying.
Once aborted, the task cannot be resumed and will be marked as aborted.

Examples:
  ampx abort abc123           # Abort task abc123
  ampx abort abc123 --force   # Force abort even if task is in progress
  ampx abort abc123 -o json   # Output result as JSON`,
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

			// Get current task status first
			task, err := getTask(client, taskID)
			if err != nil {
				return err
			}

			// Validate that task can be aborted (unless force is used)
			if !forceFlag {
				if err := validateAbortable(task); err != nil {
					return err
				}
			}

			// Confirm abort if task is running (unless force is used)
			if !forceFlag && task.Status == "running" {
				fmt.Printf("Task %s is currently running.\n", taskID)
				fmt.Printf("Repository: %s\n", task.Repo)
				fmt.Printf("Prompt: %s\n", output.TruncateString(task.Prompt, 60))
				fmt.Print("Are you sure you want to abort this task? (y/N): ")
				
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Abort cancelled.")
					return nil
				}
			}

			// Create abort request
			request := UpdateTaskRequest{
				Action: "abort",
			}

			if config.Verbose {
				fmt.Printf("Aborting task: %s\n", taskID)
			}

			// Make API request
			resp, err := client.Patch(fmt.Sprintf("/api/v1/tasks/%s", taskID), request)
			if err != nil {
				return fmt.Errorf("failed to abort task: %w", err)
			}

			// Handle response
			if err := client.HandleResponse(resp, nil); err != nil {
				return fmt.Errorf("failed to abort task: %w", err)
			}

			// Display result
			switch outputFormat {
			case "json":
				return outputAbortJSON(taskID)
			case "table", "":
				return outputAbortTable(taskID, task)
			default:
				return fmt.Errorf("unsupported output format: %s", outputFormat)
			}
		},
	}

	cmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force abort without confirmation")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// validateAbortable checks if a task can be aborted
func validateAbortable(task *TaskResponse) error {
	abortableStates := []string{"queued", "running", "retrying", "needs_review"}
	
	for _, state := range abortableStates {
		if task.Status == state {
			return nil
		}
	}

	// Special handling for already completed states
	switch task.Status {
	case "success":
		return fmt.Errorf("task has already completed successfully and cannot be aborted")
	case "failed", "error":
		return fmt.Errorf("task has already failed and cannot be aborted (use 'continue' to retry)")
	case "aborted":
		return fmt.Errorf("task is already aborted")
	default:
		return fmt.Errorf("task cannot be aborted: current status is '%s' (must be one of: %s)",
			task.Status, strings.Join(abortableStates, ", "))
	}
}

// outputAbortTable displays the result in table format
func outputAbortTable(taskID string, originalTask *TaskResponse) error {
	fmt.Println("✓ Task aborted successfully!")
	fmt.Println()
	fmt.Printf("Task ID:         %s\n", taskID)
	fmt.Printf("Previous Status: %s\n", output.Status(originalTask.Status))
	fmt.Printf("New Status:      %s\n", output.Status("aborted"))
	fmt.Printf("Repository:      %s\n", originalTask.Repo)
	fmt.Printf("Prompt:          %s\n", output.TruncateString(originalTask.Prompt, 60))
	fmt.Printf("Attempts:        %d\n", originalTask.Attempts)
	fmt.Println()
	
	switch originalTask.Status {
	case "running":
		fmt.Println("The running task has been terminated.")
		if originalTask.CIRunID != nil {
			fmt.Printf("CI run %d may still be running and should be cancelled manually.\n", *originalTask.CIRunID)
		}
	case "queued":
		fmt.Println("The task has been removed from the queue.")
	case "retrying":
		fmt.Println("The retry attempt has been cancelled.")
	case "needs_review":
		fmt.Println("The task review has been cancelled.")
	}
	
	fmt.Println()
	fmt.Println("This task cannot be resumed. Create a new task if needed:")
	fmt.Printf("  ampx start %s \"%s\"\n", originalTask.Repo, originalTask.Prompt)

	return nil
}

// outputAbortJSON displays the result in JSON format
func outputAbortJSON(taskID string) error {
	result := map[string]interface{}{
		"task_id": taskID,
		"action":  "abort",
		"status":  "success",
		"message": "Task aborted successfully",
	}
	
	return cli.PrintJSON(result)
}
