package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/models"
)

// New creates a new worker instance
func New(config *Config, taskSvc TaskService) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, config.MaxConcurrency)
	
	return &Worker{
		config:    config,
		taskSvc:   taskSvc,
		ctx:       ctx,
		cancel:    cancel,
		semaphore: semaphore,
	}
}

// Start begins the worker's main loop
func (w *Worker) Start() error {
	log.Printf("Worker starting with max concurrency: %d", w.config.MaxConcurrency)
	
	// Ensure working directory exists
	if err := os.MkdirAll(w.config.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	
	// Start the main polling loop
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			log.Println("Worker shutting down...")
			return nil
		case <-ticker.C:
			if err := w.pollForTasks(); err != nil {
				log.Printf("Error polling for tasks: %v", err)
			}
		}
	}
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() {
	log.Println("Worker stop requested")
	w.cancel()
}

// pollForTasks checks for new tasks and processes them
func (w *Worker) pollForTasks() error {
	// Try to acquire semaphore for concurrency control
	select {
	case w.semaphore <- struct{}{}:
		// Got semaphore, check for task
		task, err := w.taskSvc.GetNextTask(w.ctx)
		if err != nil {
			<-w.semaphore // Release semaphore
			return fmt.Errorf("failed to get next task: %w", err)
		}
		
		if task == nil {
			<-w.semaphore // Release semaphore, no task available
			return nil
		}
		
		// Process task in goroutine
		go w.processTask(task)
		return nil
	default:
		// All workers busy, skip this poll
		return nil
	}
}

// processTask handles execution of a single task
func (w *Worker) processTask(task *models.Task) {
	defer func() { <-w.semaphore }() // Release semaphore when done
	
	log.Printf("Processing task %d: %s", task.ID, task.Prompt)
	
	// Update task status to running
	if err := w.taskSvc.UpdateTaskStatus(w.ctx, task.ID, "running"); err != nil {
		log.Printf("Failed to update task status to running: %v", err)
		return
	}
	
	// Log task start
	w.taskSvc.AddTaskLog(w.ctx, task.ID, "info", "Task processing started")
	
	// Create task processor
	processor := &TaskProcessor{
		task:    task,
		config:  w.config,
		taskSvc: w.taskSvc,
		workDir: w.generateWorkDir(task),
	}
	
	// Execute the task
	result := processor.Execute(w.ctx)
	
	// Update task based on result
	if result.Success {
		task.Status = "completed"
		task.BranchURL = result.BranchURL
		task.PRURL = result.PRURL
		w.taskSvc.AddTaskLog(w.ctx, task.ID, "info", "Task completed successfully")
	} else {
		task.Status = "failed"
		errorMsg := "Task failed"
		if result.Error != nil {
			errorMsg = result.Error.Error()
		}
		w.taskSvc.AddTaskLog(w.ctx, task.ID, "error", errorMsg)
	}
	
	// Update task in database
	if err := w.taskSvc.UpdateTaskModel(w.ctx, task); err != nil {
		log.Printf("Failed to update task: %v", err)
	}
	
	// Clean up working directory
	if err := os.RemoveAll(processor.workDir); err != nil {
		log.Printf("Failed to clean up work directory: %v", err)
	}
	
	log.Printf("Task %d completed with status: %s", task.ID, task.Status)
}

// generateWorkDir creates a unique working directory for the task
func (w *Worker) generateWorkDir(task *models.Task) string {
	timestamp := time.Now().Format("20060102-150405")
	dirName := fmt.Sprintf("task-%d-%s", task.ID, timestamp)
	return filepath.Join(w.config.WorkDir, dirName)
}

// Execute processes the task through the complete workflow
func (tp *TaskProcessor) Execute(ctx context.Context) *ExecutionResult {
	result := &ExecutionResult{
		Success: false,
		Logs:    []string{},
	}
	
	// Step 1: Create working directory
	if err := os.MkdirAll(tp.workDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create work directory: %w", err)
		return result
	}
	
	// Step 2: Clone repository
	tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", "Cloning repository...")
	fmt.Printf("DEBUG: About to clone repository %s to %s\n", tp.task.Repo, tp.workDir)
	gitOps := NewGitOperations()
	repoDir := filepath.Join(tp.workDir, "repo")
	
	if err := gitOps.CloneRepository(ctx, tp.task.Repo, repoDir); err != nil {
		fmt.Printf("DEBUG: Clone failed: %v\n", err)
		result.Error = fmt.Errorf("failed to clone repository: %w", err)
		return result
	}
	fmt.Printf("DEBUG: Clone successful\n")
	
	// Step 3: Create feature branch
	branchName := fmt.Sprintf("amp-task-%d", tp.task.ID)
	tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", fmt.Sprintf("Creating branch: %s", branchName))
	
	if err := gitOps.CreateBranch(ctx, repoDir, branchName); err != nil {
		result.Error = fmt.Errorf("failed to create branch: %w", err)
		return result
	}
	
	// Step 4: Execute Amp prompt
	tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", "Executing Amp prompt...")
	fmt.Printf("DEBUG: About to execute Amp with prompt: %s\n", tp.task.Prompt)
	ampOps := NewAmpOperations(tp.config.AmpPath)
	
	ampResult, err := ampOps.ExecutePrompt(ctx, repoDir, tp.task.Prompt)
	if err != nil {
		fmt.Printf("DEBUG: Amp execution failed: %v\n", err)
		result.Error = fmt.Errorf("amp execution failed: %w", err)
		return result
	}
	
	fmt.Printf("DEBUG: Amp execution completed. Success: %v, Message: %s\n", ampResult.Success, ampResult.Message)
	if !ampResult.Success {
		result.Error = fmt.Errorf("amp execution unsuccessful: %s", ampResult.Message)
		return result
	}
	
	// Step 5: Commit changes
	commitMsg := fmt.Sprintf("Amp task %d: %s", tp.task.ID, truncateString(tp.task.Prompt, 50))
	tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", "Committing changes...")
	
	if err := gitOps.CommitChanges(ctx, repoDir, commitMsg); err != nil {
		result.Error = fmt.Errorf("failed to commit changes: %w", err)
		return result
	}
	
	// Step 6: Push branch
	tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", "Pushing branch...")
	
	if err := gitOps.PushBranch(ctx, repoDir, branchName); err != nil {
		result.Error = fmt.Errorf("failed to push branch: %w", err)
		return result
	}
	
	// Step 7: Create pull request (if GitHub integration is available)
	remoteURL, err := gitOps.GetRemoteURL(ctx, repoDir)
	if err == nil && tp.config.GitHubToken != "" {
		tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", "Creating pull request...")
		
		githubOps := NewGitHubOperations(tp.config.GitHubToken)
		prTitle := fmt.Sprintf("Amp Task: %s", truncateString(tp.task.Prompt, 50))
		prBody := fmt.Sprintf("Automated changes generated by Amp.\n\nOriginal prompt: %s", tp.task.Prompt)
		
		prURL, err := githubOps.CreatePullRequest(ctx, remoteURL, "main", branchName, prTitle, prBody)
		if err != nil {
			tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "warn", fmt.Sprintf("Failed to create PR: %v", err))
		} else {
			result.PRURL = prURL
			tp.taskSvc.AddTaskLog(ctx, tp.task.ID, "info", fmt.Sprintf("Pull request created: %s", prURL))
		}
	}
	
	// Generate branch URL
	if remoteURL != "" {
		result.BranchURL = fmt.Sprintf("%s/tree/%s", remoteURL, branchName)
	}
	
	result.Success = true
	result.Message = "Task completed successfully"
	return result
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
