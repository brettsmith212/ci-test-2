package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

func TestNewAbortCommand(t *testing.T) {
	cmd := NewAbortCommand()

	if cmd.Use != "abort <task-id>" {
		t.Errorf("Expected use to be 'abort <task-id>', got %s", cmd.Use)
	}

	if cmd.Short != "Abort a running or queued task" {
		t.Errorf("Expected short description to match, got %s", cmd.Short)
	}

	// Check flags exist
	flags := cmd.Flags()
	if flags.Lookup("force") == nil {
		t.Error("Expected --force flag to exist")
	}
	if flags.Lookup("output") == nil {
		t.Error("Expected --output flag to exist")
	}
}

func TestValidateAbortable(t *testing.T) {
	tests := []struct {
		name    string
		task    *TaskResponse
		wantErr bool
		errMsg  string
	}{
		{
			name: "queued task can be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "queued",
			},
			wantErr: false,
		},
		{
			name: "running task can be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "running",
			},
			wantErr: false,
		},
		{
			name: "retrying task can be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "retrying",
			},
			wantErr: false,
		},
		{
			name: "needs_review task can be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "needs_review",
			},
			wantErr: false,
		},
		{
			name: "success task cannot be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "success",
			},
			wantErr: true,
			errMsg:  "task has already completed successfully and cannot be aborted",
		},
		{
			name: "failed task cannot be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "failed",
			},
			wantErr: true,
			errMsg:  "task has already failed and cannot be aborted (use 'continue' to retry)",
		},
		{
			name: "error task cannot be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "error",
			},
			wantErr: true,
			errMsg:  "task has already failed and cannot be aborted (use 'continue' to retry)",
		},
		{
			name: "already aborted task cannot be aborted",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "aborted",
			},
			wantErr: true,
			errMsg:  "task is already aborted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAbortable(tt.task)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAbortCommandExecution(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name            string
		args            []string
		flags           map[string]string
		mockGetResp     TaskResponse
		mockGetStatus   int
		mockPatchResp   string
		mockPatchStatus int
		wantErr         bool
		errMsg          string
		checkOutput     func(t *testing.T, output string)
		checkRequest    func(t *testing.T, r *http.Request, body []byte)
		inputResponse   string // For confirmation prompts
	}{
		{
			name:  "successful abort - queued task with force",
			args:  []string{"task-123"},
			flags: map[string]string{"force": "true", "output": "table"},

			mockGetResp: TaskResponse{
				ID:        "task-123",
				Repo:      "https://github.com/user/repo.git",
				Prompt:    "Fix the authentication bug",
				Status:    "queued",
				Attempts:  1,
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now.Add(-time.Minute * 30),
			},
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task aborted"}`,
			mockPatchStatus: 200,
			wantErr:         false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "✓ Task aborted successfully!") {
					t.Error("Expected success message in output")
				}
				if !strings.Contains(output, "task-123") {
					t.Error("Expected task ID in output")
				}
				if !strings.Contains(output, "queued") {
					t.Error("Expected previous status in output")
				}
				if !strings.Contains(output, "aborted") {
					t.Error("Expected new status in output")
				}
			},
			checkRequest: func(t *testing.T, r *http.Request, body []byte) {
				var req UpdateTaskRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				if req.Action != "abort" {
					t.Errorf("Expected action 'abort', got %s", req.Action)
				}
			},
		},
		{
			name:  "successful abort - json format",
			args:  []string{"task-123"},
			flags: map[string]string{"force": "true", "output": "json"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "running",
			},
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task aborted"}`,
			mockPatchStatus: 200,
			wantErr:         false,
			checkOutput: func(t *testing.T, output string) {
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(output), &response); err != nil {
					t.Errorf("Expected valid JSON output, got error: %v", err)
				}
				if response["task_id"] != "task-123" {
					t.Errorf("Expected task_id in JSON output")
				}
			},
		},
		{
			name:           "task not found",
			args:           []string{"nonexistent"},
			flags:          map[string]string{"force": "true", "output": "table"},
			mockGetResp:    TaskResponse{},
			mockGetStatus:  404,
			mockPatchResp:  "",
			mockPatchStatus: 0,
			wantErr:        true,
			errMsg:         "failed to get task",
		},
		{
			name:  "task cannot be aborted - success",
			args:  []string{"task-123"},
			flags: map[string]string{"force": "true", "output": "table"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "success",
			},
			mockGetStatus:   200,
			mockPatchResp:   "",
			mockPatchStatus: 0,
			wantErr:         true,
			errMsg:          "Task has already completed successfully and cannot be aborted",
		},
		{
			name:  "task cannot be aborted - failed",
			args:  []string{"task-123"},
			flags: map[string]string{"force": "true", "output": "table"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "failed",
			},
			mockGetStatus:   200,
			mockPatchResp:   "",
			mockPatchStatus: 0,
			wantErr:         true,
			errMsg:          "Task has already failed. Use 'continue' to retry or 'start' to create a new task",
		},
		{
			name:            "API error during abort",
			args:            []string{"task-123"},
			flags:           map[string]string{"force": "true", "output": "table"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "running",
			},
			mockGetStatus:   200,
			mockPatchResp:   `{"error": "internal server error"}`,
			mockPatchStatus: 500,
			wantErr:         true,
			errMsg:          "failed to abort task",
		},
		{
			name:           "missing task ID argument",
			args:           []string{},
			flags:          map[string]string{"force": "true", "output": "table"},
			mockGetResp:    TaskResponse{},
			mockGetStatus:  0,
			mockPatchResp:  "",
			mockPatchStatus: 0,
			wantErr:        true,
			errMsg:         "accepts 1 arg(s), received 0",
		},
		{
			name:            "invalid output format",
			args:            []string{"task-123"},
			flags:           map[string]string{"force": "true", "output": "xml"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "running",
			},
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task aborted"}`,
			mockPatchStatus: 200,
			wantErr:         true,
			errMsg:          "unsupported output format: xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track API calls
			var getCallCount, patchCallCount int

			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					getCallCount++
					w.WriteHeader(tt.mockGetStatus)
					if tt.mockGetStatus == 200 {
						json.NewEncoder(w).Encode(tt.mockGetResp)
					} else {
						w.Write([]byte(`{"error": "task not found"}`))
					}
				} else if r.Method == "PATCH" {
					patchCallCount++
					
					// Read request body
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("Failed to read request body: %v", err)
					}

					if tt.checkRequest != nil {
						tt.checkRequest(t, r, body)
					}

					w.WriteHeader(tt.mockPatchStatus)
					w.Write([]byte(tt.mockPatchResp))
				}
			}))
			defer mockServer.Close()

			// Create command
			cmd := NewAbortCommand()

			// Set flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}

			// Set API URL
			cmd.Flags().Set("api-url", mockServer.URL)
			
			// Set parent flags for config loading
			cmd.Flags().String("verbose", "false", "verbose")
			cmd.Flags().Set("verbose", "false")

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set args
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}

				// Verify API calls were made (should have GET call, and PATCH call if task is abortable)
				if getCallCount == 0 {
					t.Error("Expected GET call to be made")
				}

				// Check output
				if tt.checkOutput != nil {
					tt.checkOutput(t, buf.String())
				}
			}
		})
	}
}

func TestOutputAbortTable(t *testing.T) {
	now := time.Now()
	originalTask := &TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		Prompt:    "Fix the authentication bug",
		Status:    "running",
		Attempts:  1,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Minute * 30),
	}

	tests := []struct {
		name         string
		taskID       string
		originalTask *TaskResponse
		expected     []string
	}{
		{
			name:         "successful abort output",
			taskID:       "task-123",
			originalTask: originalTask,
			expected: []string{
				"✓ Task aborted successfully!",
				"task-123",
				"running",
				"aborted",
				"https://github.com/user/repo.git",
				"Fix the authentication bug",
				"The task has been aborted and will not continue processing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			
			// Temporarily redirect output
			oldOutput := cli.GetOutput()
			cli.SetOutput(&buf)
			defer cli.SetOutput(oldOutput)

			err := outputAbortTable(tt.taskID, tt.originalTask)
			if err != nil {
				t.Fatalf("outputAbortTable failed: %v", err)
			}

			output := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestOutputAbortJSON(t *testing.T) {
	taskID := "task-123"

	var buf bytes.Buffer
	
	// Temporarily redirect output
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	err := outputAbortJSON(taskID)
	if err != nil {
		t.Fatalf("outputAbortJSON failed: %v", err)
	}

	output := buf.String()
	
	// Verify it's valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	// Check JSON content
	if response["task_id"] != taskID {
		t.Errorf("Expected task_id %s, got %v", taskID, response["task_id"])
	}
	if response["action"] != "abort" {
		t.Errorf("Expected action 'abort', got %v", response["action"])
	}
	if response["status"] != "aborted" {
		t.Errorf("Expected status 'aborted', got %v", response["status"])
	}
}

// Test status-specific abort messages
func TestAbortStatusMessages(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		status   string
		expected []string
	}{
		{
			name:   "running task abort message",
			status: "running",
			expected: []string{
				"The task was actively being processed by Amp and has been stopped",
			},
		},
		{
			name:   "queued task abort message",
			status: "queued",
			expected: []string{
				"The task has been removed from the processing queue",
			},
		},
		{
			name:   "retrying task abort message",
			status: "retrying",
			expected: []string{
				"The task retry has been cancelled",
			},
		},
		{
			name:   "needs_review task abort message",
			status: "needs_review",
			expected: []string{
				"The task review has been cancelled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalTask := &TaskResponse{
				ID:        "task-123",
				Repo:      "https://github.com/user/repo.git",
				Prompt:    "Fix the authentication bug",
				Status:    tt.status,
				CreatedAt: now,
				UpdatedAt: now,
			}

			var buf bytes.Buffer
			
			// Temporarily redirect output
			oldOutput := cli.GetOutput()
			cli.SetOutput(&buf)
			defer cli.SetOutput(oldOutput)

			err := outputAbortTable("task-123", originalTask)
			if err != nil {
				t.Fatalf("outputAbortTable failed: %v", err)
			}

			output := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s' for status %s, got:\n%s", expected, tt.status, output)
				}
			}
		})
	}
}

// Test force flag
func TestForceFlag(t *testing.T) {
	cmd := NewAbortCommand()
	
	// Set force flag
	cmd.Flags().Set("force", "true")
	
	// Check that force flag is properly set
	forceFlag, err := cmd.Flags().GetBool("force")
	if err != nil {
		t.Fatalf("Failed to get force flag: %v", err)
	}
	
	if !forceFlag {
		t.Error("Expected force flag to be true")
	}
}

// Test confirmation logic (would need to be mocked in real tests)
func TestConfirmationLogic(t *testing.T) {
	// This test would be more complex in a real scenario where we need to mock stdin
	// For now, we just test that the logic exists
	
	cmd := NewAbortCommand()
	
	// Without force flag, running tasks should require confirmation
	forceFlag, _ := cmd.Flags().GetBool("force")
	if forceFlag {
		t.Error("Expected force flag to be false by default")
	}
}

// Test edge cases
func TestAbortEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		task    *TaskResponse
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil task",
			task:    nil,
			wantErr: true,
			errMsg:  "task is nil",
		},
		{
			name: "task with empty status",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "",
			},
			wantErr: true,
			errMsg:  "invalid task status",
		},
		{
			name: "task with unknown status",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "unknown_status",
			},
			wantErr: true,
			errMsg:  "invalid task status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAbortable(tt.task)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateAbortable(b *testing.B) {
	task := &TaskResponse{
		ID:     "task-123",
		Status: "running",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateAbortable(task)
	}
}

func BenchmarkOutputAbortTable(b *testing.B) {
	now := time.Now()
	originalTask := &TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		Prompt:    "Fix the authentication bug",
		Status:    "running",
		Attempts:  1,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Minute * 30),
	}

	var buf bytes.Buffer
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		outputAbortTable("task-123", originalTask)
	}
}
