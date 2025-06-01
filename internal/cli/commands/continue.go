package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
	"github.com/brettsmith212/ci-test-2/internal/cli/output"
)

// UpdateTaskRequest represents a task update request
type UpdateTaskRequest struct {
	Action string `json:"action"`
	Prompt string `json:"prompt,omitempty"`
}

// NewContinueCommand creates the continue command
func NewContinueCommand() *cobra.Command {
	var waitFlag bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "continue <task-id> [new-prompt]",
		Short: "Continue a failed or paused task",
		Long: `Continue a failed or paused task, optionally with a modified prompt.

This command can be used to:
- Retry a failed task with the same prompt
- Retry a failed task with a modified prompt for better results
- Resume a task that needs review

Examples:
  ampx continue abc123                                    # Retry with same prompt
  ampx continue abc123 "Try a different approach"        # Retry with new prompt
  ampx continue abc123 "Focus on error handling" --wait  # Retry and wait for completion`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			var newPrompt string
			if len(args) > 1 {
				newPrompt = args[1]
			}

			// Load configuration
			config, err := cli.LoadConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create client
			client := cli.NewClient(config)

			// Validate new prompt if provided
			if newPrompt != "" {
				if err := validatePrompt(newPrompt); err != nil {
					return fmt.Errorf("invalid prompt: %w", err)
				}
			}

			// Get current task status first
			task, err := getTask(client, taskID)
			if err != nil {
				return err
			}

			// Validate that task can be continued
			if err := validateContinuable(task); err != nil {
				return err
			}

			// Create update request
			request := UpdateTaskRequest{
				Action: "continue",
				Prompt: newPrompt,
			}

			if config.Verbose {
				fmt.Printf("Continuing task: %s\n", taskID)
				if newPrompt != "" {
					fmt.Printf("New prompt: %s\n", newPrompt)
				} else {
					fmt.Println("Using original prompt")
				}
			}

			// Make API request
			resp, err := client.Patch(fmt.Sprintf("/api/v1/tasks/%s", taskID), request)
			if err != nil {
				return fmt.Errorf("failed to continue task: %w", err)
			}

			// Handle response
			if err := client.HandleResponse(resp, nil); err != nil {
				return fmt.Errorf("failed to continue task: %w", err)
			}

			// Display result
			switch outputFormat {
			case "json":
				return outputContinueJSON(taskID, newPrompt)
			case "table", "":
				return outputContinueTable(taskID, newPrompt, task)
			default:
				return fmt.Errorf("unsupported output format: %s", outputFormat)
			}
		},
	}

	cmd.Flags().BoolVarP(&waitFlag, "wait", "w", false, "Wait for task completion before returning")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// getTask fetches a task by ID
func getTask(client *cli.Client, taskID string) (*TaskResponse, error) {
	resp, err := client.Get(fmt.Sprintf("/api/v1/tasks/%s", taskID))
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task TaskResponse
	if err := client.HandleResponse(resp, &task); err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &task, nil
}

// validateContinuable checks if a task can be continued
func validateContinuable(task *TaskResponse) error {
	continuableStates := []string{"failed", "error", "retrying", "needs_review"}
	
	for _, state := range continuableStates {
		if task.Status == state {
			return nil
		}
	}

	return fmt.Errorf("task cannot be continued: current status is '%s' (must be one of: %s)",
		task.Status, strings.Join(continuableStates, ", "))
}

// validatePrompt validates the new prompt
func validatePrompt(prompt string) error {
	if len(prompt) < 10 {
		return fmt.Errorf("prompt must be at least 10 characters long")
	}

	if len(prompt) > 1000 {
		return fmt.Errorf("prompt cannot exceed 1000 characters")
	}

	// Check for potentially dangerous content
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"rm -rf",
		"sudo rm",
		"eval(",
		"exec(",
	}

	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return fmt.Errorf("prompt contains potentially dangerous content: %s", pattern)
		}
	}

	return nil
}

// outputContinueTable displays the result in table format
func outputContinueTable(taskID, newPrompt string, originalTask *TaskResponse) error {
	fmt.Println("✓ Task continued successfully!")
	fmt.Println()
	fmt.Printf("Task ID:        %s\n", taskID)
	fmt.Printf("Previous Status: %s\n", output.Status(originalTask.Status))
	fmt.Printf("New Status:     %s\n", output.Status("queued"))
	fmt.Printf("Repository:     %s\n", originalTask.Repo)
	
	if newPrompt != "" {
		fmt.Printf("Original Prompt: %s\n", output.TruncateString(originalTask.Prompt, 60))
		fmt.Printf("New Prompt:     %s\n", newPrompt)
	} else {
		fmt.Printf("Prompt:         %s\n", output.TruncateString(originalTask.Prompt, 60))
	}
	
	fmt.Printf("Attempts:       %d → %d\n", originalTask.Attempts, originalTask.Attempts+1)
	fmt.Println()
	fmt.Println("The task has been queued for retry.")
	fmt.Println("Use 'ampx logs " + taskID + "' to monitor progress")
	fmt.Println("Use 'ampx list --status=running' to see active tasks")

	return nil
}

// outputContinueJSON displays the result in JSON format
func outputContinueJSON(taskID, newPrompt string) error {
	result := map[string]interface{}{
		"task_id":    taskID,
		"action":     "continue",
		"status":     "success",
		"message":    "Task continued successfully",
	}
	
	if newPrompt != "" {
		result["new_prompt"] = newPrompt
	}
	
	return cli.PrintJSON(result)
}
