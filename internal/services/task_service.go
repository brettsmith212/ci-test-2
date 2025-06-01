package services

import (
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/brettsmith212/ci-test-2/internal/database"
	"github.com/brettsmith212/ci-test-2/internal/models"
)

// TaskService provides business logic for task operations
type TaskService struct {
	db *gorm.DB
}

// NewTaskService creates a new TaskService instance
func NewTaskService() *TaskService {
	db := database.GetDB()
	if db == nil {
		panic("database not initialized - call database.Connect() first")
	}
	return &TaskService{
		db: db,
	}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(repo, prompt string) (*models.Task, error) {
	// Generate unique ID
	id := ulid.Make().String()
	
	// Generate branch name from ID
	branch := fmt.Sprintf("amp/%s", id[:6])
	
	// TODO: Generate Amp thread ID
	// For now, use a placeholder - this will be implemented in worker step
	threadID := fmt.Sprintf("thread-%s", id[:8])

	task := &models.Task{
		ID:       id,
		Repo:     repo,
		Branch:   branch,
		ThreadID: threadID,
		Prompt:   prompt,
		Status:   models.TaskStatusQueued,
		Attempts: 0,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(id string) (*models.Task, error) {
	var task models.Task
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to retrieve task: %w", err)
	}

	return &task, nil
}

// ListTasks retrieves tasks with optional filtering
func (s *TaskService) ListTasks(status string, limit, offset int) ([]models.Task, error) {
	var tasks []models.Task
	query := s.db.Model(&models.Task{})

	// Apply status filter if provided
	if status != "" {
		// Validate status
		taskStatus := models.TaskStatus(status)
		if !taskStatus.IsValid() {
			return nil, fmt.Errorf("invalid status: %s", status)
		}
		query = query.Where("status = ?", status)
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Order by most recent first
	query = query.Order("created_at DESC")

	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, nil
}

// UpdateTask updates a task based on action
func (s *TaskService) UpdateTask(id, action, prompt string) error {
	// Retrieve the task
	task, err := s.GetTask(id)
	if err != nil {
		return err
	}

	switch action {
	case "continue":
		// Validate that task can be continued
		if !task.IsRetryable(3) { // TODO: Get max retries from config
			return fmt.Errorf("task cannot be continued: status=%s, attempts=%d", task.Status, task.Attempts)
		}

		// Update prompt if provided
		if prompt != "" {
			task.Prompt = prompt
		}

		// Update status to queued for retry
		if err := task.UpdateStatus(models.TaskStatusQueued); err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

	case "abort":
		// Validate that task can be aborted
		if task.Status.IsTerminal() && task.Status != models.TaskStatusAborted {
			// Allow aborting already completed tasks (idempotent)
			if task.Status == models.TaskStatusSuccess {
				return nil // Already completed, nothing to abort
			}
		}

		// Update status to aborted
		if err := task.UpdateStatus(models.TaskStatusAborted); err != nil {
			return fmt.Errorf("failed to abort task: %w", err)
		}

	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	// Save the updated task
	if err := s.db.Save(task).Error; err != nil {
		return fmt.Errorf("failed to save updated task: %w", err)
	}

	return nil
}

// GetTasksByRepo retrieves tasks for a specific repository
func (s *TaskService) GetTasksByRepo(repo string, limit, offset int) ([]models.Task, error) {
	var tasks []models.Task
	query := s.db.Where("repo = ?", repo)

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Order by most recent first
	query = query.Order("created_at DESC")

	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks by repo: %w", err)
	}

	return tasks, nil
}

// GetActiveTasks retrieves all non-terminal tasks
func (s *TaskService) GetActiveTasks() ([]models.Task, error) {
	var tasks []models.Task
	
	// Get tasks that are not in terminal states
	query := s.db.Where("status IN ?", []string{
		string(models.TaskStatusQueued),
		string(models.TaskStatusRunning),
		string(models.TaskStatusRetrying),
		string(models.TaskStatusNeedsReview),
	})

	if err := query.Order("created_at ASC").Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}

	return tasks, nil
}

// ValidateRepo validates repository format
func (s *TaskService) ValidateRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repo cannot be empty")
	}

	// Basic validation for Git repository format
	if !strings.Contains(repo, "/") {
		return fmt.Errorf("repo must be in format 'owner/repo' or full Git URL")
	}

	// Additional validation can be added here
	return nil
}

// ValidatePrompt validates prompt content
func (s *TaskService) ValidatePrompt(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	if len(prompt) > 10000 { // Reasonable limit
		return fmt.Errorf("prompt too long (max 10000 characters)")
	}

	return nil
}
