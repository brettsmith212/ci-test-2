package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
	"github.com/brettsmith212/ci-test-2/internal/cli/output"
)

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Repo   string `json:"repo"`
	Prompt string `json:"prompt"`
}

// CreateTaskResponse represents a task creation response
type CreateTaskResponse struct {
	ID     string `json:"id"`
	Branch string `json:"branch"`
}

// NewStartCommand creates the start command
func NewStartCommand() *cobra.Command {
	var waitFlag bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "start <repository> <prompt>",
		Short: "Start a new CI-driven Amp task",
		Long: `Start a new CI-driven Amp task for the specified repository with the given prompt.

The repository should be a valid Git URL (GitHub, GitLab, Bitbucket supported).
The prompt should describe what you want Amp to do.

Examples:
  ampx start https://github.com/user/repo.git "Fix the authentication bug"
  ampx start git@github.com:user/repo.git "Add unit tests for user service"
  ampx start --wait https://github.com/user/repo.git "Optimize database queries"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]
			prompt := args[1]

			// Load configuration
			config, err := cli.LoadConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create client
			client := cli.NewClient(config)

			// Validate inputs
			if err := validateStartInputs(repo, prompt); err != nil {
				return err
			}

			// Create task request
			request := CreateTaskRequest{
				Repo:   repo,
				Prompt: prompt,
			}

			if config.Verbose {
				fmt.Printf("Creating task for repository: %s\n", repo)
				fmt.Printf("Prompt: %s\n", prompt)
			}

			// Make API request
			resp, err := client.Post("/api/v1/tasks", request)
			if err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			// Parse response
			var createResp CreateTaskResponse
			if err := client.HandleResponse(resp, &createResp); err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			// Display result based on format
			switch outputFormat {
			case "json":
				return outputJSON(createResp)
			case "table", "":
				return outputStartTable(createResp, repo, prompt)
			default:
				return fmt.Errorf("unsupported output format: %s", outputFormat)
			}
		},
	}

	cmd.Flags().BoolVarP(&waitFlag, "wait", "w", false, "Wait for task completion before returning")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// validateStartInputs validates the repository URL and prompt
func validateStartInputs(repo, prompt string) error {
	// Validate repository URL
	if repo == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Basic URL validation
	validPrefixes := []string{
		"https://github.com/",
		"https://gitlab.com/",
		"https://bitbucket.org/",
		"git@github.com:",
		"git@gitlab.com:",
		"git@bitbucket.org:",
	}

	valid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(repo, prefix) {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("repository URL must be a valid Git URL (GitHub, GitLab, or Bitbucket)")
	}

	// Validate prompt
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

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

// outputStartTable displays the result in table format
func outputStartTable(resp CreateTaskResponse, repo, prompt string) error {
	output.PrintSuccess("Task created successfully!")
	fmt.Println()
	
	fmt.Printf("%-12s %s\n", output.Primary("Task ID:"), output.ID(resp.ID))
	fmt.Printf("%-12s %s\n", output.Primary("Branch:"), output.Branch(resp.Branch))
	fmt.Printf("%-12s %s\n", output.Primary("Repository:"), output.Repository(repo))
	fmt.Printf("%-12s %s\n", output.Primary("Prompt:"), prompt)
	
	fmt.Println()
	fmt.Printf("%s %s\n", output.Info("Use"), output.Code("ampx logs "+resp.ID)+" to monitor progress")
	fmt.Printf("%s %s\n", output.Info("Use"), output.Code("ampx list")+" to see all tasks")

	return nil
}

// outputJSON displays the result in JSON format
func outputJSON(data interface{}) error {
	return cli.PrintJSON(data)
}
