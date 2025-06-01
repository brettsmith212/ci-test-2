package handlers

import (
	"time"

	"github.com/brettsmith212/ci-test-2/internal/models"
)

// CreateTaskRequest represents the request payload for creating a new task
type CreateTaskRequest struct {
	Repo   string `json:"repo" binding:"required"`
	Prompt string `json:"prompt" binding:"required"`
}

// CreateTaskResponse represents the response after creating a task
type CreateTaskResponse struct {
	ID     string `json:"id"`
	Branch string `json:"branch"`
}

// UpdateTaskRequest represents the request payload for updating a task
type UpdateTaskRequest struct {
	Action string `json:"action" binding:"required,oneof=continue abort"`
	Prompt string `json:"prompt,omitempty"`
}

// TaskResponse represents a task in API responses
type TaskResponse struct {
	ID        string                `json:"id"`
	Repo      string                `json:"repo"`
	Branch    string                `json:"branch,omitempty"`
	ThreadID  string                `json:"thread_id,omitempty"`
	Prompt    string                `json:"prompt"`
	Status    models.TaskStatus     `json:"status"`
	CIRunID   *int64                `json:"ci_run_id,omitempty"`
	Attempts  int                   `json:"attempts"`
	Summary   string                `json:"summary,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// TaskListResponse represents the response for listing tasks
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// ValidationErrorResponse represents validation error details
type ValidationErrorResponse struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// ToTaskResponse converts a models.Task to TaskResponse
func ToTaskResponse(task *models.Task) TaskResponse {
	return TaskResponse{
		ID:        task.ID,
		Repo:      task.Repo,
		Branch:    task.Branch,
		ThreadID:  task.ThreadID,
		Prompt:    task.Prompt,
		Status:    task.Status,
		CIRunID:   task.CIRunID,
		Attempts:  task.Attempts,
		Summary:   task.Summary,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
}

// ToTaskListResponse converts a slice of models.Task to TaskListResponse
func ToTaskListResponse(tasks []models.Task) TaskListResponse {
	taskResponses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = ToTaskResponse(&task)
	}

	return TaskListResponse{
		Tasks: taskResponses,
		Total: len(tasks),
	}
}
