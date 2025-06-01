package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitOperations implements the GitOperations interface
type gitOperations struct{}

// NewGitOperations creates a new Git operations instance
func NewGitOperations() GitOperations {
	return &gitOperations{}
}

// CloneRepository clones a Git repository to the specified destination
func (g *gitOperations) CloneRepository(ctx context.Context, repoURL, destDir string) error {
	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(destDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	
	// Add timeout to prevent hanging
	cloneCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	
	// Execute git clone
	cmd := exec.CommandContext(cloneCtx, "git", "clone", repoURL, destDir)
	
	// Set up environment
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", // Disable interactive prompts
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w (output: %s)", err, string(output))
	}
	
	return nil
}

// CreateBranch creates and checks out a new branch
func (g *gitOperations) CreateBranch(ctx context.Context, repoDir, branchName string) error {
	// Validate branch name
	if err := g.validateBranchName(branchName); err != nil {
		return err
	}
	
	// Create and checkout the branch
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	cmd.Dir = repoDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w (output: %s)", branchName, err, string(output))
	}
	
	return nil
}

// CommitChanges adds all changes and commits them with the specified message
func (g *gitOperations) CommitChanges(ctx context.Context, repoDir, message string) error {
	// First, add all changes
	addCmd := exec.CommandContext(ctx, "git", "add", ".")
	addCmd.Dir = repoDir
	
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w (output: %s)", err, string(output))
	}
	
	// Check if there are changes to commit
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = repoDir
	
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return fmt.Errorf("no changes to commit")
	}
	
	// Commit the changes
	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commitCmd.Dir = repoDir
	
	// Configure git user if not already set
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Amp Worker",
		"GIT_AUTHOR_EMAIL=amp-worker@example.com",
		"GIT_COMMITTER_NAME=Amp Worker",
		"GIT_COMMITTER_EMAIL=amp-worker@example.com",
	)
	
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w (output: %s)", err, string(output))
	}
	
	return nil
}

// PushBranch pushes the specified branch to the remote repository
func (g *gitOperations) PushBranch(ctx context.Context, repoDir, branchName string) error {
	// Set upstream and push
	pushCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(pushCtx, "git", "push", "-u", "origin", branchName)
	cmd.Dir = repoDir
	
	// Set up environment to disable interactive prompts
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %w (output: %s)", err, string(output))
	}
	
	return nil
}

// GetRemoteURL retrieves the remote URL of the repository
func (g *gitOperations) GetRemoteURL(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	
	url := strings.TrimSpace(string(output))
	
	// Convert SSH URLs to HTTPS for web viewing
	url = g.normalizeGitURL(url)
	
	return url, nil
}

// GetCurrentBranch returns the name of the current branch
func (g *gitOperations) GetCurrentBranch(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	
	return strings.TrimSpace(string(output)), nil
}

// HasChanges checks if there are uncommitted changes in the repository
func (g *gitOperations) HasChanges(ctx context.Context, repoDir string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetLastCommitHash returns the hash of the last commit
func (g *gitOperations) GetLastCommitHash(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}
	
	return strings.TrimSpace(string(output)), nil
}

// ConfigureRepository sets up basic git configuration for the repository
func (g *gitOperations) ConfigureRepository(ctx context.Context, repoDir string) error {
	configs := map[string]string{
		"user.name":  "Amp Worker",
		"user.email": "amp-worker@example.com",
	}
	
	for key, value := range configs {
		cmd := exec.CommandContext(ctx, "git", "config", key, value)
		cmd.Dir = repoDir
		
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set git config %s: %w (output: %s)", key, err, string(output))
		}
	}
	
	return nil
}

// validateBranchName ensures the branch name is valid for Git
func (g *gitOperations) validateBranchName(branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	
	// Check for invalid characters
	invalidChars := []string{" ", "~", "^", ":", "?", "*", "[", "\\", "..", "@{"}
	for _, char := range invalidChars {
		if strings.Contains(branchName, char) {
			return fmt.Errorf("branch name contains invalid character: %s", char)
		}
	}
	
	// Check for invalid starting/ending characters
	if strings.HasPrefix(branchName, "-") || strings.HasSuffix(branchName, "-") {
		return fmt.Errorf("branch name cannot start or end with hyphen")
	}
	
	if strings.HasPrefix(branchName, ".") || strings.HasSuffix(branchName, ".") {
		return fmt.Errorf("branch name cannot start or end with dot")
	}
	
	return nil
}

// normalizeGitURL converts SSH URLs to HTTPS for web viewing
func (g *gitOperations) normalizeGitURL(url string) string {
	// Convert git@github.com:user/repo.git to https://github.com/user/repo
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		return "https://github.com/" + path
	}
	
	// Convert git@gitlab.com:user/repo.git to https://gitlab.com/user/repo
	if strings.HasPrefix(url, "git@gitlab.com:") {
		path := strings.TrimPrefix(url, "git@gitlab.com:")
		path = strings.TrimSuffix(path, ".git")
		return "https://gitlab.com/" + path
	}
	
	// Remove .git suffix from HTTPS URLs
	if strings.HasSuffix(url, ".git") {
		return strings.TrimSuffix(url, ".git")
	}
	
	return url
}

// CleanupRepository performs cleanup operations on the repository
func (g *gitOperations) CleanupRepository(ctx context.Context, repoDir string) error {
	// Clean untracked files
	cleanCmd := exec.CommandContext(ctx, "git", "clean", "-fd")
	cleanCmd.Dir = repoDir
	
	if output, err := cleanCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clean failed: %w (output: %s)", err, string(output))
	}
	
	// Reset any changes
	resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", "HEAD")
	resetCmd.Dir = repoDir
	
	if output, err := resetCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset failed: %w (output: %s)", err, string(output))
	}
	
	return nil
}
