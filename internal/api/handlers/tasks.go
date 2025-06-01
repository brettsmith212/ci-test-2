package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brettsmith212/ci-test-2/internal/services"
	"github.com/brettsmith212/ci-test-2/internal/validation"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	taskService *services.TaskService
}

// NewTaskHandler creates a new TaskHandler instance
func NewTaskHandler() *TaskHandler {
	// Create the service once when the handler is created
	taskService := services.NewTaskService()
	return &TaskHandler{
		taskService: taskService,
	}
}

// CreateTask handles POST /tasks
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrs := validation.TranslateValidationErrors(err)
		c.JSON(http.StatusBadRequest, ValidationErrorResponse{
			Error:     "validation_error",
			Message:   "Request validation failed",
			Fields:    map[string]string{"validation": validationErrs.Error()},
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate repository format using new validator
	if err := validation.ValidateRepositoryURL(req.Repo); err != nil {
		c.JSON(http.StatusBadRequest, ValidationErrorResponse{
			Error:     "validation_error",
			Message:   "Invalid repository",
			Fields:    map[string]string{"repo": err.Error()},
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate prompt using new validator
	if err := validation.ValidatePromptContent(req.Prompt); err != nil {
		c.JSON(http.StatusBadRequest, ValidationErrorResponse{
			Error:     "validation_error",
			Message:   "Invalid prompt",
			Fields:    map[string]string{"prompt": err.Error()},
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Create the task
	task, err := h.taskService.CreateTask(req.Repo, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "creation_error",
			Message:   "Failed to create task",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Return success response
	response := CreateTaskResponse{
		ID:     task.ID,
		Branch: task.Branch,
	}

	c.JSON(http.StatusCreated, response)
}

// GetTask handles GET /tasks/{id}
func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "validation_error",
			Message:   "Task ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	task, err := h.taskService.GetTask(id)
	if err != nil {
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:     "not_found",
				Message:   "Task not found",
				RequestID: c.GetString("request_id"),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "retrieval_error",
			Message:   "Failed to retrieve task",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	response := ToTaskResponse(task)
	c.JSON(http.StatusOK, response)
}

// ListTasks handles GET /tasks
func (h *TaskHandler) ListTasks(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "validation_error",
			Message:   "Invalid limit parameter",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "validation_error",
			Message:   "Invalid offset parameter",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Apply reasonable limits
	if limit > 100 {
		limit = 100
	}

	tasks, err := h.taskService.ListTasks(status, limit, offset)
	if err != nil {
		if err.Error() == "invalid status: "+status {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "validation_error",
				Message:   err.Error(),
				RequestID: c.GetString("request_id"),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "retrieval_error",
			Message:   "Failed to retrieve tasks",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	response := ToTaskListResponse(tasks)
	c.JSON(http.StatusOK, response)
}

// UpdateTask handles PATCH /tasks/{id}
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "validation_error",
			Message:   "Task ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ValidationErrorResponse{
			Error:     "validation_error",
			Message:   "Invalid request payload",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate prompt if action is continue and prompt is provided
	if req.Action == "continue" && req.Prompt != "" {
		if err := h.taskService.ValidatePrompt(req.Prompt); err != nil {
			c.JSON(http.StatusBadRequest, ValidationErrorResponse{
				Error:     "validation_error",
				Message:   err.Error(),
				RequestID: c.GetString("request_id"),
			})
			return
		}
	}

	// Update the task
	err := h.taskService.UpdateTask(id, req.Action, req.Prompt)
	if err != nil {
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:     "not_found",
				Message:   "Task not found",
				RequestID: c.GetString("request_id"),
			})
			return
		}

		// Check for business logic errors
		if err.Error() == "task cannot be continued: status=success, attempts=3" ||
		   err.Error() == "failed to update task status: invalid value" {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:     "conflict",
				Message:   err.Error(),
				RequestID: c.GetString("request_id"),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "update_error",
			Message:   "Failed to update task",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Return 204 No Content for successful updates
	c.Status(http.StatusNoContent)
}

// GetActiveTasksHandler handles GET /tasks/active
func (h *TaskHandler) GetActiveTasks(c *gin.Context) {
	tasks, err := h.taskService.GetActiveTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "retrieval_error",
			Message:   "Failed to retrieve active tasks",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	response := ToTaskListResponse(tasks)
	c.JSON(http.StatusOK, response)
}

// GetTasksByRepoHandler handles GET /tasks?repo={repo}
func (h *TaskHandler) GetTasksByRepo(c *gin.Context) {
	repo := c.Query("repo")
	if repo == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "validation_error",
			Message:   "Repository parameter is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	tasks, err := h.taskService.GetTasksByRepo(repo, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "retrieval_error",
			Message:   "Failed to retrieve tasks for repository",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	response := ToTaskListResponse(tasks)
	c.JSON(http.StatusOK, response)
}
