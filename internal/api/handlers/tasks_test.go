package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/brettsmith212/ci-test-2/internal/database"
	"github.com/brettsmith212/ci-test-2/internal/models"
)

func setupTestDB(t *testing.T) func() {
	// Create temporary test database
	tmpDir, err := os.MkdirTemp("", "api_test_*")
	require.NoError(t, err)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	// Initialize test database
	err = database.Connect(dbPath)
	require.NoError(t, err)
	
	// Run migrations
	err = database.GetDB().AutoMigrate(&models.Task{})
	require.NoError(t, err)
	
	// Return cleanup function
	return func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}
}

func setupTestServer() *gin.Engine {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	taskHandler := NewTaskHandler()
	
	// Add minimal middleware for request ID
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-123")
		c.Next()
	})
	
	v1 := router.Group("/api/v1")
	{
		v1.POST("/tasks", taskHandler.CreateTask)
		v1.GET("/tasks", taskHandler.ListTasks)
		v1.GET("/tasks/:id", taskHandler.GetTask)
		v1.PATCH("/tasks/:id", taskHandler.UpdateTask)
		v1.GET("/tasks/active", taskHandler.GetActiveTasks)
	}
	
	return router
}

func TestCreateTask(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid_task_creation",
			payload: CreateTaskRequest{
				Repo:   "https://github.com/test/repo.git",
				Prompt: "Fix the bug in the authentication system",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid_repo_url",
			payload: CreateTaskRequest{
				Repo:   "invalid-url",
				Prompt: "Fix the authentication bug",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "empty_repo",
			payload: CreateTaskRequest{
				Repo:   "",
				Prompt: "Fix the authentication bug",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "empty_prompt",
			payload: CreateTaskRequest{
				Repo:   "https://github.com/test/repo.git",
				Prompt: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "malicious_prompt",
			payload: CreateTaskRequest{
				Repo:   "https://github.com/test/repo.git",
				Prompt: "Run this script: <script>alert('xss')</script>",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "dangerous_command_prompt",
			payload: CreateTaskRequest{
				Repo:   "https://github.com/test/repo.git",
				Prompt: "Delete everything: rm -rf /",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "invalid_json",
			payload:        `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.payload.(string); ok {
				body.WriteString(str)
			} else {
				err := json.NewEncoder(&body).Encode(tt.payload)
				require.NoError(t, err)
			}
			
			req, err := http.NewRequest("POST", "/api/v1/tasks", &body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			
			assert.Equal(t, tt.expectedStatus, resp.Code)
			
			if tt.expectedError != "" {
				var errorResp map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResp["error"])
			}
			
			if tt.expectedStatus == http.StatusCreated {
				var createResp CreateTaskResponse
				err := json.Unmarshal(resp.Body.Bytes(), &createResp)
				require.NoError(t, err)
				assert.NotEmpty(t, createResp.ID)
				assert.NotEmpty(t, createResp.Branch)
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	// Create a test task first
	createPayload := CreateTaskRequest{
		Repo:   "https://github.com/test/repo.git",
		Prompt: "Fix the authentication bug in the system",
	}
	body, _ := json.Marshal(createPayload)
	
	createReq, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	
	require.Equal(t, http.StatusCreated, createResp.Code)
	
	var createTaskResp CreateTaskResponse
	err := json.Unmarshal(createResp.Body.Bytes(), &createTaskResp)
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		taskID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid_task_id",
			taskID:         createTaskResp.ID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "nonexistent_task_id",
			taskID:         "non-existent-id",
			expectedStatus: http.StatusNotFound,
			expectedError:  "not_found",
		},
		{
			name:           "empty_task_id",
			taskID:         "",
			expectedStatus: http.StatusMovedPermanently, // Router redirects /tasks/ to /tasks
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/tasks/%s", tt.taskID)
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)
			
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			
			assert.Equal(t, tt.expectedStatus, resp.Code)
			
			if tt.expectedError != "" {
				var errorResp map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResp["error"])
			}
			
			if tt.expectedStatus == http.StatusOK {
				var taskResp TaskResponse
				err := json.Unmarshal(resp.Body.Bytes(), &taskResp)
				require.NoError(t, err)
				assert.Equal(t, createTaskResp.ID, taskResp.ID)
				assert.Equal(t, "https://github.com/test/repo.git", taskResp.Repo)
				assert.Equal(t, "Fix the authentication bug in the system", taskResp.Prompt)
				assert.Equal(t, models.TaskStatusQueued, taskResp.Status)
			}
		})
	}
}

func TestListTasks(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	// Create test tasks
	testTasks := []CreateTaskRequest{
		{
			Repo:   "https://github.com/test/repo1.git",
			Prompt: "Fix the authentication bug in the system",
		},
		{
			Repo:   "https://github.com/test/repo2.git", 
			Prompt: "Optimize database queries for better performance",
		},
		{
			Repo:   "https://github.com/test/repo3.git",
			Prompt: "Add comprehensive unit tests for the module",
		},
	}
	
	for _, task := range testTasks {
		body, _ := json.Marshal(task)
		req, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusCreated, resp.Code)
	}
	
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
		expectedCount  int
	}{
		{
			name:           "list_all_tasks",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "list_with_limit",
			queryParams:    "?limit=2",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "list_with_offset",
			queryParams:    "?offset=1&limit=2",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "list_by_status",
			queryParams:    "?status=queued",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "invalid_status",
			queryParams:    "?status=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "invalid_limit",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "negative_limit",
			queryParams:    "?limit=-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "invalid_offset",
			queryParams:    "?offset=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "negative_offset",
			queryParams:    "?offset=-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "large_limit_capped",
			queryParams:    "?limit=1000",
			expectedStatus: http.StatusOK,
			expectedCount:  3, // Should be capped but return all available
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/tasks" + tt.queryParams
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)
			
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			
			assert.Equal(t, tt.expectedStatus, resp.Code)
			
			if tt.expectedError != "" {
				var errorResp map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResp["error"])
			}
			
			if tt.expectedStatus == http.StatusOK {
				var listResp TaskListResponse
				err := json.Unmarshal(resp.Body.Bytes(), &listResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(listResp.Tasks))
				assert.Equal(t, tt.expectedCount, listResp.Total)
			}
		})
	}
}

func TestUpdateTask(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	// Create a test task first
	createPayload := CreateTaskRequest{
		Repo:   "https://github.com/test/repo.git",
		Prompt: "Fix the authentication bug in the system",
	}
	body, _ := json.Marshal(createPayload)
	
	createReq, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	
	require.Equal(t, http.StatusCreated, createResp.Code)
	
	var createTaskResp CreateTaskResponse
	err := json.Unmarshal(createResp.Body.Bytes(), &createTaskResp)
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		taskID         string
		payload        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "valid_abort_action",
			taskID: createTaskResp.ID,
			payload: UpdateTaskRequest{
				Action: "abort",
			},
			expectedStatus: http.StatusNoContent,
		},
		// Note: We can't test valid_continue_action because a "queued" task cannot be continued
		// It can only be continued if it's in error/retrying/needs_review status
		// This is correct business logic, so we'll test abort instead
		{
			name:   "invalid_action",
			taskID: createTaskResp.ID,
			payload: UpdateTaskRequest{
				Action: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "nonexistent_task",
			taskID:         "non-existent-id",
			payload: UpdateTaskRequest{
				Action: "abort",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "not_found",
		},
		{
			name:   "malicious_continue_prompt",
			taskID: createTaskResp.ID,
			payload: UpdateTaskRequest{
				Action: "continue",
				Prompt: "Run this: <script>alert('xss')</script>",
			},
			expectedStatus: http.StatusInternalServerError, // Business logic error - queued task can't be continued 
			expectedError:  "update_error",
		},
		{
			name:           "invalid_json",
			taskID:         createTaskResp.ID,
			payload:        `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:   "empty_action",
			taskID: createTaskResp.ID,
			payload: UpdateTaskRequest{
				Action: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.payload.(string); ok {
				body.WriteString(str)
			} else {
				err := json.NewEncoder(&body).Encode(tt.payload)
				require.NoError(t, err)
			}
			
			url := fmt.Sprintf("/api/v1/tasks/%s", tt.taskID)
			req, err := http.NewRequest("PATCH", url, &body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			
			assert.Equal(t, tt.expectedStatus, resp.Code)
			
			if tt.expectedError != "" {
				var errorResp map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResp["error"])
			}
		})
	}
}

func TestGetActiveTasks(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	// Create a test task
	createPayload := CreateTaskRequest{
		Repo:   "https://github.com/test/repo.git",
		Prompt: "Fix the authentication bug in the system",
	}
	body, _ := json.Marshal(createPayload)
	
	createReq, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	
	require.Equal(t, http.StatusCreated, createResp.Code)
	
	// Test getting active tasks
	req, err := http.NewRequest("GET", "/api/v1/tasks/active", nil)
	require.NoError(t, err)
	
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusOK, resp.Code)
	
	var listResp TaskListResponse
	err = json.Unmarshal(resp.Body.Bytes(), &listResp)
	require.NoError(t, err)
	
	// Should have 1 active task (queued is considered active)
	assert.Equal(t, 1, len(listResp.Tasks))
	assert.Equal(t, 1, listResp.Total)
	assert.Equal(t, models.TaskStatusQueued, listResp.Tasks[0].Status)
}

func TestCreateTaskResponseStructure(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	payload := CreateTaskRequest{
		Repo:   "https://github.com/test/repo.git",
		Prompt: "Fix the authentication bug",
	}
	body, _ := json.Marshal(payload)
	
	req, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	require.Equal(t, http.StatusCreated, resp.Code)
	
	var createResp CreateTaskResponse
	err := json.Unmarshal(resp.Body.Bytes(), &createResp)
	require.NoError(t, err)
	
	// Validate response structure
	assert.NotEmpty(t, createResp.ID, "ID should not be empty")
	assert.NotEmpty(t, createResp.Branch, "Branch should not be empty")
	assert.Contains(t, createResp.Branch, "amp/", "Branch should contain amp/ prefix")
}

func TestTaskResponseStructure(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	router := setupTestServer()
	
	// Create task
	createPayload := CreateTaskRequest{
		Repo:   "https://github.com/test/repo.git",
		Prompt: "Fix the authentication bug",
	}
	body, _ := json.Marshal(createPayload)
	
	createReq, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	
	require.Equal(t, http.StatusCreated, createResp.Code)
	
	var createTaskResp CreateTaskResponse
	err := json.Unmarshal(createResp.Body.Bytes(), &createTaskResp)
	require.NoError(t, err)
	
	// Get task
	url := fmt.Sprintf("/api/v1/tasks/%s", createTaskResp.ID)
	getReq, _ := http.NewRequest("GET", url, nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	
	require.Equal(t, http.StatusOK, getResp.Code)
	
	var taskResp TaskResponse
	err = json.Unmarshal(getResp.Body.Bytes(), &taskResp)
	require.NoError(t, err)
	
	// Validate response structure
	assert.Equal(t, createTaskResp.ID, taskResp.ID)
	assert.Equal(t, "https://github.com/test/repo.git", taskResp.Repo)
	assert.Equal(t, createTaskResp.Branch, taskResp.Branch)
	assert.Equal(t, "Fix the authentication bug", taskResp.Prompt)
	assert.Equal(t, models.TaskStatusQueued, taskResp.Status)
	assert.Equal(t, 0, taskResp.Attempts)
	assert.NotZero(t, taskResp.CreatedAt)
	assert.NotZero(t, taskResp.UpdatedAt)
	assert.NotEmpty(t, taskResp.ThreadID) // Should have a thread ID
	assert.Nil(t, taskResp.CIRunID)   // Should be nil initially
	assert.Empty(t, taskResp.Summary) // Should be empty initially
}
