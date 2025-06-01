package worker

import (
	"context"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/models"
)

// Config holds worker configuration
type Config struct {
	// Polling interval for checking new tasks
	PollInterval time.Duration
	// Maximum number of concurrent tasks
	MaxConcurrency int
	// Working directory for repositories
	WorkDir string
	// Amp CLI binary path
	AmpPath string
	// GitHub token for API access
	GitHubToken string
	// Database configuration
	DatabasePath string
}

// Worker represents a task processing worker
type Worker struct {
	config   *Config
	taskSvc  TaskService
	ctx      context.Context
	cancel   context.CancelFunc
	semaphore chan struct{}
}

// TaskService interface for task operations
type TaskService interface {
	GetNextTask(ctx context.Context) (*models.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status string) error
	UpdateTaskModel(ctx context.Context, task *models.Task) error
	AddTaskLog(ctx context.Context, taskID string, level, message string) error
}

// TaskProcessor handles individual task execution
type TaskProcessor struct {
	task    *models.Task
	config  *Config
	taskSvc TaskService
	workDir string
}

// ExecutionResult represents the result of task execution
type ExecutionResult struct {
	Success   bool
	Message   string
	BranchURL string
	PRURL     string
	Logs      []string
	Error     error
}

// GitOperations interface for Git operations
type GitOperations interface {
	CloneRepository(ctx context.Context, repoURL, destDir string) error
	CreateBranch(ctx context.Context, repoDir, branchName string) error
	CommitChanges(ctx context.Context, repoDir, message string) error
	PushBranch(ctx context.Context, repoDir, branchName string) error
	GetRemoteURL(ctx context.Context, repoDir string) (string, error)
}

// AmpOperations interface for Amp CLI operations
type AmpOperations interface {
	ExecutePrompt(ctx context.Context, repoDir, prompt string) (*AmpResult, error)
	CheckInstallation() error
}

// AmpResult represents the result of Amp execution
type AmpResult struct {
	Success     bool
	Message     string
	FilesChanged []string
	Output      string
	Error       error
}

// GitHubOperations interface for GitHub API operations
type GitHubOperations interface {
	CreatePullRequest(ctx context.Context, repoURL, baseBranch, headBranch, title, body string) (string, error)
	GetPullRequestStatus(ctx context.Context, prURL string) (string, error)
	GetWorkflowRuns(ctx context.Context, repoURL, branchName string) ([]WorkflowRun, error)
}

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID         int64
	Status     string
	Conclusion string
	HTMLURL    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
