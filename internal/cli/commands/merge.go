package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

// NewMergeCommand creates the merge command
func NewMergeCommand() *cobra.Command {
	var autoFlag bool
	var outputFormat string
	var deleteFlag bool

	cmd := &cobra.Command{
		Use:   "merge <task-id>",
		Short: "Merge a successfully completed task",
		Long: `Merge a successfully completed task's changes.

This command is used to merge the changes from a successful task.
It provides information about the branch and pull request associated
with the task, and guidance on how to merge the changes.

Note: This command currently provides guidance for manual merging.
Automatic merging may be implemented in future versions.

Examples:
  ampx merge abc123              # Get merge information for task abc123
  ampx merge abc123 --auto       # Auto-merge (not yet implemented)
  ampx merge abc123 -o json      # Output merge info as JSON`,
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

			// Get current task status
			task, err := getTask(client, taskID)
			if err != nil {
				return err
			}

			// Validate that task can be merged
			if err := validateMergeable(task); err != nil {
				return err
			}

			// For now, we just provide merge information
			// TODO: Implement actual merging logic when worker is implemented
			if autoFlag {
				return fmt.Errorf("automatic merging is not yet implemented")
			}

			// Display merge information
			switch outputFormat {
			case "json":
				return outputMergeJSON(task)
			case "table", "":
				return outputMergeTable(task)
			default:
				return fmt.Errorf("unsupported output format: %s", outputFormat)
			}
		},
	}

	cmd.Flags().BoolVarP(&autoFlag, "auto", "a", false, "Automatically merge the PR (not yet implemented)")
	cmd.Flags().BoolVar(&deleteFlag, "delete-branch", false, "Delete the branch after merging (not yet implemented)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

// validateMergeable checks if a task can be merged
func validateMergeable(task *TaskResponse) error {
	if task.Status != "success" {
		return fmt.Errorf("task cannot be merged: current status is '%s' (must be 'success')", task.Status)
	}

	if task.Branch == "" {
		return fmt.Errorf("task has no associated branch")
	}

	return nil
}

// outputMergeTable displays merge information in table format
func outputMergeTable(task *TaskResponse) error {
	fmt.Println("✓ Task ready for merge!")
	fmt.Println()
	fmt.Printf("Task ID:     %s\n", task.ID)
	fmt.Printf("Status:      %s\n", formatStatus(task.Status))
	fmt.Printf("Repository:  %s\n", task.Repo)
	fmt.Printf("Branch:      %s\n", task.Branch)
	if task.ThreadID != "" {
		fmt.Printf("Thread ID:   %s\n", task.ThreadID)
	}
	if task.CIRunID != nil {
		fmt.Printf("CI Run:      %d (passed)\n", *task.CIRunID)
	}
	fmt.Printf("Prompt:      %s\n", task.Prompt)

	if task.Summary != "" {
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(task.Summary)
	}

	fmt.Println()
	fmt.Println("Merge Instructions:")
	fmt.Println(strings.Repeat("=", 50))

	// Extract repository information
	repoURL := task.Repo
	
	// Remove .git suffix if present for web URLs
	if strings.HasSuffix(repoURL, ".git") {
		repoURL = strings.TrimSuffix(repoURL, ".git")
	}
	
	// Convert SSH URLs to HTTPS for web viewing
	if strings.HasPrefix(repoURL, "git@") {
		// Convert git@github.com:user/repo to https://github.com/user/repo
		parts := strings.Split(repoURL, ":")
		if len(parts) == 2 {
			domain := strings.TrimPrefix(parts[0], "git@")
			repoURL = fmt.Sprintf("https://%s/%s", domain, parts[1])
		}
	}

	fmt.Println("1. Review the changes:")
	fmt.Printf("   Branch: %s\n", task.Branch)
	if strings.Contains(repoURL, "github.com") {
		fmt.Printf("   Compare: %s/compare/%s\n", repoURL, task.Branch)
	}

	fmt.Println()
	fmt.Println("2. Create a Pull Request (if not already created):")
	if strings.Contains(repoURL, "github.com") {
		fmt.Printf("   GitHub: %s/compare/%s\n", repoURL, task.Branch)
	} else if strings.Contains(repoURL, "gitlab.com") {
		fmt.Printf("   GitLab: %s/-/merge_requests/new?merge_request[source_branch]=%s\n", repoURL, task.Branch)
	} else if strings.Contains(repoURL, "bitbucket.org") {
		fmt.Printf("   Bitbucket: %s/pull-requests/new?source=%s\n", repoURL, task.Branch)
	} else {
		fmt.Printf("   Create PR from branch: %s\n", task.Branch)
	}

	fmt.Println()
	fmt.Println("3. Merge via web interface or command line:")
	fmt.Println("   Web: Use the merge button in your pull request")
	fmt.Println("   CLI: ")
	fmt.Printf("     git checkout main\n")
	fmt.Printf("     git pull origin main\n")
	fmt.Printf("     git merge %s\n", task.Branch)
	fmt.Printf("     git push origin main\n")
	fmt.Printf("     git branch -d %s\n", task.Branch)
	fmt.Printf("     git push origin --delete %s\n", task.Branch)

	fmt.Println()
	fmt.Println("4. Optional cleanup:")
	fmt.Printf("   Delete remote branch: git push origin --delete %s\n", task.Branch)

	fmt.Println()
	fmt.Printf("📝 Task completed: %s\n", task.Prompt)
	fmt.Printf("🎉 Ready to merge changes from %s\n", task.Branch)

	return nil
}

// outputMergeJSON displays merge information in JSON format
func outputMergeJSON(task *TaskResponse) error {
	repoURL := task.Repo
	if strings.HasSuffix(repoURL, ".git") {
		repoURL = strings.TrimSuffix(repoURL, ".git")
	}
	
	// Convert SSH URLs to HTTPS
	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) == 2 {
			domain := strings.TrimPrefix(parts[0], "git@")
			repoURL = fmt.Sprintf("https://%s/%s", domain, parts[1])
		}
	}

	mergeInfo := map[string]interface{}{
		"task_id":    task.ID,
		"status":     task.Status,
		"repository": task.Repo,
		"branch":     task.Branch,
		"prompt":     task.Prompt,
		"summary":    task.Summary,
		"merge_info": map[string]interface{}{
			"ready_to_merge": true,
			"branch_name":    task.Branch,
			"repository_url": repoURL,
		},
	}

	if task.CIRunID != nil {
		mergeInfo["ci_run_id"] = *task.CIRunID
	}

	// Add platform-specific URLs
	if strings.Contains(repoURL, "github.com") {
		mergeInfo["merge_info"].(map[string]interface{})["compare_url"] = fmt.Sprintf("%s/compare/%s", repoURL, task.Branch)
		mergeInfo["merge_info"].(map[string]interface{})["pr_url"] = fmt.Sprintf("%s/compare/%s", repoURL, task.Branch)
	} else if strings.Contains(repoURL, "gitlab.com") {
		mergeInfo["merge_info"].(map[string]interface{})["mr_url"] = fmt.Sprintf("%s/-/merge_requests/new?merge_request[source_branch]=%s", repoURL, task.Branch)
	} else if strings.Contains(repoURL, "bitbucket.org") {
		mergeInfo["merge_info"].(map[string]interface{})["pr_url"] = fmt.Sprintf("%s/pull-requests/new?source=%s", repoURL, task.Branch)
	}

	return cli.PrintJSON(mergeInfo)
}

// extractRepoName extracts repository name from URL
func extractRepoName(repoURL string) string {
	// Remove .git suffix
	if strings.HasSuffix(repoURL, ".git") {
		repoURL = strings.TrimSuffix(repoURL, ".git")
	}
	
	// Extract name from URL
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return repoURL
}
