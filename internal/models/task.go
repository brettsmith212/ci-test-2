package models

import (
	"time"

	"gorm.io/gorm"
)

// TaskStatus represents the possible states of a task
type TaskStatus string

const (
	TaskStatusQueued      TaskStatus = "queued"
	TaskStatusRunning     TaskStatus = "running"
	TaskStatusRetrying    TaskStatus = "retrying"
	TaskStatusNeedsReview TaskStatus = "needs_review"
	TaskStatusSuccess     TaskStatus = "success"
	TaskStatusAborted     TaskStatus = "aborted"
	TaskStatusError       TaskStatus = "error"
)

// IsValid checks if the task status is valid
func (ts TaskStatus) IsValid() bool {
	switch ts {
	case TaskStatusQueued, TaskStatusRunning, TaskStatusRetrying,
		 TaskStatusNeedsReview, TaskStatusSuccess, TaskStatusAborted, TaskStatusError:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status indicates the task is finished
func (ts TaskStatus) IsTerminal() bool {
	switch ts {
	case TaskStatusSuccess, TaskStatusAborted, TaskStatusError:
		return true
	default:
		return false
	}
}

// Task represents a CI-driven Amp task
type Task struct {
	ID         string     `gorm:"primaryKey;type:text" json:"id"`
	Repo       string     `gorm:"not null;type:text" json:"repo"`
	Branch     string     `gorm:"type:text" json:"branch"`
	ThreadID   string     `gorm:"type:text" json:"thread_id"`
	Prompt     string     `gorm:"type:text" json:"prompt"`
	Status     TaskStatus `gorm:"type:text;not null;default:'queued'" json:"status"`
	CIRunID    *int64     `gorm:"type:integer" json:"ci_run_id,omitempty"`
	Attempts   int        `gorm:"type:integer;default:0" json:"attempts"`
	Summary    string     `gorm:"type:text" json:"summary,omitempty"`
	BranchURL  string     `gorm:"type:text" json:"branch_url,omitempty"`
	PRURL      string     `gorm:"type:text" json:"pr_url,omitempty"`
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// BeforeCreate is a GORM hook that runs before creating a task
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	// Validate status
	if !t.Status.IsValid() {
		t.Status = TaskStatusQueued
	}
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a task
func (t *Task) BeforeUpdate(tx *gorm.DB) error {
	// Validate status if it's being updated
	if !t.Status.IsValid() {
		return gorm.ErrInvalidValue
	}
	return nil
}



// CanTransitionTo checks if the task can transition to the given status
func (t *Task) CanTransitionTo(newStatus TaskStatus) bool {
	// If task is already in a terminal state, only allow transition to aborted
	if t.Status.IsTerminal() {
		return newStatus == TaskStatusAborted
	}

	// Define valid transitions
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusQueued: {
			TaskStatusRunning,
			TaskStatusAborted,
		},
		TaskStatusRunning: {
			TaskStatusRetrying,
			TaskStatusNeedsReview,
			TaskStatusSuccess,
			TaskStatusError,
			TaskStatusAborted,
		},
		TaskStatusRetrying: {
			TaskStatusRunning,
			TaskStatusNeedsReview,
			TaskStatusError,
			TaskStatusAborted,
		},
		TaskStatusNeedsReview: {
			TaskStatusRunning,
			TaskStatusAborted,
		},
	}

	allowedStatuses, exists := validTransitions[t.Status]
	if !exists {
		return false
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// UpdateStatus updates the task status if the transition is valid
func (t *Task) UpdateStatus(newStatus TaskStatus) error {
	if !t.CanTransitionTo(newStatus) {
		return gorm.ErrInvalidValue
	}
	t.Status = newStatus
	return nil
}

// IncrementAttempts increments the attempt counter
func (t *Task) IncrementAttempts() {
	t.Attempts++
}

// IsRetryable returns true if the task can be retried
func (t *Task) IsRetryable(maxRetries int) bool {
	return t.Attempts < maxRetries && 
		   (t.Status == TaskStatusError || t.Status == TaskStatusRetrying || t.Status == TaskStatusNeedsReview)
}

// TaskLog represents a log entry for a task
type TaskLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"not null;index;type:text" json:"task_id"`
	Level     string    `gorm:"not null" json:"level"` // info, warn, error
	Message   string    `gorm:"not null" json:"message"`
	Timestamp time.Time `gorm:"not null" json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
