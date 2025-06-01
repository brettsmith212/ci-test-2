package models

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		status TaskStatus
		valid  bool
	}{
		{TaskStatusQueued, true},
		{TaskStatusRunning, true},
		{TaskStatusRetrying, true},
		{TaskStatusNeedsReview, true},
		{TaskStatusSuccess, true},
		{TaskStatusAborted, true},
		{TaskStatusError, true},
		{TaskStatus("invalid"), false},
		{TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("TaskStatus.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestTaskStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		terminal bool
	}{
		{TaskStatusQueued, false},
		{TaskStatusRunning, false},
		{TaskStatusRetrying, false},
		{TaskStatusNeedsReview, false},
		{TaskStatusSuccess, true},
		{TaskStatusAborted, true},
		{TaskStatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("TaskStatus.IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}

func TestTask_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  TaskStatus
		toStatus    TaskStatus
		canTransition bool
	}{
		// From queued
		{"queued to running", TaskStatusQueued, TaskStatusRunning, true},
		{"queued to aborted", TaskStatusQueued, TaskStatusAborted, true},
		{"queued to success", TaskStatusQueued, TaskStatusSuccess, false},
		
		// From running
		{"running to retrying", TaskStatusRunning, TaskStatusRetrying, true},
		{"running to needs_review", TaskStatusRunning, TaskStatusNeedsReview, true},
		{"running to success", TaskStatusRunning, TaskStatusSuccess, true},
		{"running to error", TaskStatusRunning, TaskStatusError, true},
		{"running to aborted", TaskStatusRunning, TaskStatusAborted, true},
		{"running to queued", TaskStatusRunning, TaskStatusQueued, false},
		
		// From retrying
		{"retrying to running", TaskStatusRetrying, TaskStatusRunning, true},
		{"retrying to needs_review", TaskStatusRetrying, TaskStatusNeedsReview, true},
		{"retrying to error", TaskStatusRetrying, TaskStatusError, true},
		{"retrying to aborted", TaskStatusRetrying, TaskStatusAborted, true},
		{"retrying to success", TaskStatusRetrying, TaskStatusSuccess, false},
		
		// From needs_review
		{"needs_review to running", TaskStatusNeedsReview, TaskStatusRunning, true},
		{"needs_review to aborted", TaskStatusNeedsReview, TaskStatusAborted, true},
		{"needs_review to success", TaskStatusNeedsReview, TaskStatusSuccess, false},
		
		// From terminal states
		{"success to aborted", TaskStatusSuccess, TaskStatusAborted, true},
		{"success to running", TaskStatusSuccess, TaskStatusRunning, false},
		{"error to aborted", TaskStatusError, TaskStatusAborted, true},
		{"error to running", TaskStatusError, TaskStatusRunning, false},
		{"aborted to running", TaskStatusAborted, TaskStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{Status: tt.fromStatus}
			if got := task.CanTransitionTo(tt.toStatus); got != tt.canTransition {
				t.Errorf("Task.CanTransitionTo() = %v, want %v", got, tt.canTransition)
			}
		})
	}
}

func TestTask_UpdateStatus(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  TaskStatus
		toStatus    TaskStatus
		expectError bool
	}{
		{"valid transition", TaskStatusQueued, TaskStatusRunning, false},
		{"invalid transition", TaskStatusQueued, TaskStatusSuccess, true},
		{"terminal to aborted", TaskStatusSuccess, TaskStatusAborted, false},
		{"terminal to running", TaskStatusSuccess, TaskStatusRunning, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{Status: tt.fromStatus}
			err := task.UpdateStatus(tt.toStatus)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Task.UpdateStatus() expected error, got nil")
				}
				// Status should not change on error
				if task.Status != tt.fromStatus {
					t.Errorf("Task.UpdateStatus() changed status on error: got %v, want %v", task.Status, tt.fromStatus)
				}
			} else {
				if err != nil {
					t.Errorf("Task.UpdateStatus() unexpected error: %v", err)
				}
				// Status should change on success
				if task.Status != tt.toStatus {
					t.Errorf("Task.UpdateStatus() status not updated: got %v, want %v", task.Status, tt.toStatus)
				}
			}
		})
	}
}

func TestTask_IncrementAttempts(t *testing.T) {
	task := &Task{Attempts: 0}
	
	task.IncrementAttempts()
	if task.Attempts != 1 {
		t.Errorf("Task.IncrementAttempts() = %d, want 1", task.Attempts)
	}
	
	task.IncrementAttempts()
	if task.Attempts != 2 {
		t.Errorf("Task.IncrementAttempts() = %d, want 2", task.Attempts)
	}
}

func TestTask_IsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		attempts   int
		status     TaskStatus
		maxRetries int
		retryable  bool
	}{
		{"error with attempts left", 1, TaskStatusError, 3, true},
		{"error at max attempts", 3, TaskStatusError, 3, false},
		{"retrying with attempts left", 2, TaskStatusRetrying, 3, true},
		{"needs_review with attempts left", 1, TaskStatusNeedsReview, 3, true},
		{"success not retryable", 1, TaskStatusSuccess, 3, false},
		{"queued not retryable", 0, TaskStatusQueued, 3, false},
		{"running not retryable", 1, TaskStatusRunning, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Attempts: tt.attempts,
				Status:   tt.status,
			}
			
			if got := task.IsRetryable(tt.maxRetries); got != tt.retryable {
				t.Errorf("Task.IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestTask_BeforeCreate(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  TaskStatus
		expectedStatus TaskStatus
	}{
		{"valid status unchanged", TaskStatusRunning, TaskStatusRunning},
		{"invalid status set to queued", TaskStatus("invalid"), TaskStatusQueued},
		{"empty status set to queued", TaskStatus(""), TaskStatusQueued},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{Status: tt.initialStatus}
			
			// Call BeforeCreate hook
			err := task.BeforeCreate(&gorm.DB{})
			if err != nil {
				t.Errorf("Task.BeforeCreate() unexpected error: %v", err)
			}
			
			if task.Status != tt.expectedStatus {
				t.Errorf("Task.BeforeCreate() status = %v, want %v", task.Status, tt.expectedStatus)
			}
		})
	}
}

func TestTask_BeforeUpdate(t *testing.T) {
	tests := []struct {
		name        string
		status      TaskStatus
		expectError bool
	}{
		{"valid status", TaskStatusRunning, false},
		{"invalid status", TaskStatus("invalid"), true},
		{"empty status", TaskStatus(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{Status: tt.status}
			
			err := task.BeforeUpdate(&gorm.DB{})
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Task.BeforeUpdate() expected error, got nil")
				}
				if err != gorm.ErrInvalidValue {
					t.Errorf("Task.BeforeUpdate() expected ErrInvalidValue, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Task.BeforeUpdate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestTask_DefaultValues(t *testing.T) {
	task := &Task{
		ID:     "test-id",
		Repo:   "test/repo",
		Prompt: "test prompt",
	}

	// Test that default values are properly set
	if task.Status == "" {
		task.Status = TaskStatusQueued
	}
	
	if task.Attempts != 0 {
		t.Errorf("Task.Attempts default = %d, want 0", task.Attempts)
	}
	
	if task.Status != TaskStatusQueued {
		t.Errorf("Task.Status default = %v, want %v", task.Status, TaskStatusQueued)
	}
}

func TestTask_FieldTypes(t *testing.T) {
	now := time.Now()
	ciRunID := int64(12345)
	
	task := &Task{
		ID:        "01H1234567890ABCDEF",
		Repo:      "github.com/test/repo",
		Branch:    "amp/abc123",
		ThreadID:  "thread-123",
		Prompt:    "Fix the failing tests",
		Status:    TaskStatusRunning,
		CIRunID:   &ciRunID,
		Attempts:  2,
		Summary:   "Task in progress",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test that all fields are properly set
	if task.ID != "01H1234567890ABCDEF" {
		t.Errorf("Task.ID = %v, want %v", task.ID, "01H1234567890ABCDEF")
	}
	
	if task.CIRunID == nil || *task.CIRunID != 12345 {
		t.Errorf("Task.CIRunID = %v, want %v", task.CIRunID, 12345)
	}
	
	if task.Attempts != 2 {
		t.Errorf("Task.Attempts = %v, want %v", task.Attempts, 2)
	}
}
