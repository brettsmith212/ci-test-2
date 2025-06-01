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

func TestNewContinueCommand(t *testing.T) {
	cmd := NewContinueCommand()

	if cmd.Use != "continue <task-id> [new-prompt]" {
		t.Errorf("Expected use to be 'continue <task-id> [new-prompt]', got %s", cmd.Use)
	}

	if cmd.Short != "Continue a failed or paused task" {
		t.Errorf("Expected short description to match, got %s", cmd.Short)
	}

	// Check flags exist
	flags := cmd.Flags()
	if flags.Lookup("wait") == nil {
		t.Error("Expected --wait flag to exist")
	}
	if flags.Lookup("output") == nil {
		t.Error("Expected --output flag to exist")
	}
}

func TestValidateContinuable(t *testing.T) {
	tests := []struct {
		name    string
		task    *TaskResponse
		wantErr bool
		errMsg  string
	}{
		{
			name: "failed task can be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "failed",
			},
			wantErr: false,
		},
		{
			name: "error task can be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "error",
			},
			wantErr: false,
		},
		{
			name: "needs_review task can be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "needs_review",
			},
			wantErr: false,
		},
		{
			name: "retrying task can be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "retrying",
			},
			wantErr: false,
		},
		{
			name: "aborted task can be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "aborted",
			},
			wantErr: false,
		},
		{
			name: "running task cannot be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "running",
			},
			wantErr: true,
			errMsg:  "Task is currently running and cannot be continued",
		},
		{
			name: "queued task cannot be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "queued",
			},
			wantErr: true,
			errMsg:  "Task is already queued for processing",
		},
		{
			name: "success task cannot be continued",
			task: &TaskResponse{
				ID:     "task-123",
				Status: "success",
			},
			wantErr: true,
			errMsg:  "Task has already completed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContinuable(tt.task)

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

func TestContinueCommandExecution(t *testing.T) {
	now := time.Now()
	sampleTask := TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		ThreadID:  "thread-123",
		Prompt:    "Fix the authentication bug",
		Status:    "failed",
		CIRunID:   nil,
		Attempts:  2,
		Summary:   "Previous attempt failed due to timeout",
		CreatedAt: now.Add(-time.Hour * 2),
		UpdatedAt: now.Add(-time.Hour),
	}

	tests := []struct {
		name           string
		args           []string
		flags          map[string]string
		mockGetResp    TaskResponse
		mockGetStatus  int
		mockPatchResp  string
		mockPatchStatus int
		wantErr        bool
		errMsg         string
		checkOutput    func(t *testing.T, output string)
		checkRequest   func(t *testing.T, r *http.Request, body []byte)
	}{
		{
			name:            "successful continue without new prompt - table format",
			args:            []string{"task-123"},
			flags:           map[string]string{"output": "table"},
			mockGetResp:     sampleTask,
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task continued"}`,
			mockPatchStatus: 200,
			wantErr:         false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "✓ Task continued successfully!") {
					t.Error("Expected success message in output")
				}
				if !strings.Contains(output, "task-123") {
					t.Error("Expected task ID in output")
				}
				if !strings.Contains(output, "failed") {
					t.Error("Expected previous status in output")
				}
				if !strings.Contains(output, "queued") {
					t.Error("Expected new status in output")
				}
			},
			checkRequest: func(t *testing.T, r *http.Request, body []byte) {
				var req UpdateTaskRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				if req.Action != "continue" {
					t.Errorf("Expected action 'continue', got %s", req.Action)
				}
				if req.Prompt != "" {
					t.Errorf("Expected empty prompt, got %s", req.Prompt)
				}
			},
		},
		{
			name:            "successful continue with new prompt",
			args:            []string{"task-123", "Try a different approach to fix the bug"},
			flags:           map[string]string{"output": "table"},
			mockGetResp:     sampleTask,
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task continued"}`,
			mockPatchStatus: 200,
			wantErr:         false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Try a different approach to fix the bug") {
					t.Error("Expected new prompt in output")
				}
				if !strings.Contains(output, "Fix the authentication bug") {
					t.Error("Expected original prompt in output")
				}
			},
			checkRequest: func(t *testing.T, r *http.Request, body []byte) {
				var req UpdateTaskRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				if req.Action != "continue" {
					t.Errorf("Expected action 'continue', got %s", req.Action)
				}
				if req.Prompt != "Try a different approach to fix the bug" {
					t.Errorf("Expected new prompt, got %s", req.Prompt)
				}
			},
		},
		{
			name:            "successful continue - json format",
			args:            []string{"task-123"},
			flags:           map[string]string{"output": "json"},
			mockGetResp:     sampleTask,
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task continued"}`,
			mockPatchStatus: 200,
			wantErr:         false,
			checkOutput: func(t *testing.T, output string) {
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(output), &response); err != nil {
					t.Errorf("Expected valid JSON output, got error: %v", err)
				}
			},
		},
		{
			name:           "task not found",
			args:           []string{"nonexistent"},
			flags:          map[string]string{"output": "table"},
			mockGetResp:    TaskResponse{},
			mockGetStatus:  404,
			mockPatchResp:  "",
			mockPatchStatus: 0,
			wantErr:        true,
			errMsg:         "failed to get task",
		},
		{
			name: "task cannot be continued - running",
			args: []string{"task-123"},
			flags: map[string]string{"output": "table"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "running",
			},
			mockGetStatus:   200,
			mockPatchResp:   "",
			mockPatchStatus: 0,
			wantErr:         true,
			errMsg:          "Task is currently running and cannot be continued",
		},
		{
			name: "task cannot be continued - success",
			args: []string{"task-123"},
			flags: map[string]string{"output": "table"},
			mockGetResp: TaskResponse{
				ID:     "task-123",
				Status: "success",
			},
			mockGetStatus:   200,
			mockPatchResp:   "",
			mockPatchStatus: 0,
			wantErr:         true,
			errMsg:          "Task has already completed successfully",
		},
		{
			name:            "API error during continue",
			args:            []string{"task-123"},
			flags:           map[string]string{"output": "table"},
			mockGetResp:     sampleTask,
			mockGetStatus:   200,
			mockPatchResp:   `{"error": "internal server error"}`,
			mockPatchStatus: 500,
			wantErr:         true,
			errMsg:          "failed to continue task",
		},
		{
			name:           "missing task ID argument",
			args:           []string{},
			flags:          map[string]string{"output": "table"},
			mockGetResp:    TaskResponse{},
			mockGetStatus:  0,
			mockPatchResp:  "",
			mockPatchStatus: 0,
			wantErr:        true,
			errMsg:         "accepts between 1 and 2 arg(s), received 0",
		},
		{
			name:           "too many arguments",
			args:           []string{"task-123", "prompt1", "prompt2"},
			flags:          map[string]string{"output": "table"},
			mockGetResp:    TaskResponse{},
			mockGetStatus:  0,
			mockPatchResp:  "",
			mockPatchStatus: 0,
			wantErr:        true,
			errMsg:         "accepts between 1 and 2 arg(s), received 3",
		},
		{
			name:            "invalid output format",
			args:            []string{"task-123"},
			flags:           map[string]string{"output": "xml"},
			mockGetResp:     sampleTask,
			mockGetStatus:   200,
			mockPatchResp:   `{"message": "Task continued"}`,
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
			cmd := NewContinueCommand()

			// Set flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}

			// Set API URL
			cmd.Flags().Set("api-url", mockServer.URL)

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

				// Verify API calls were made
				if getCallCount == 0 {
					t.Error("Expected GET call to be made")
				}
				if patchCallCount == 0 {
					t.Error("Expected PATCH call to be made")
				}

				// Check output
				if tt.checkOutput != nil {
					tt.checkOutput(t, buf.String())
				}
			}
		})
	}
}

func TestOutputContinueTable(t *testing.T) {
	now := time.Now()
	originalTask := &TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		Prompt:    "Fix the authentication bug",
		Status:    "failed",
		Attempts:  2,
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Minute * 30),
	}

	tests := []struct {
		name         string
		taskID       string
		newPrompt    string
		originalTask *TaskResponse
		expected     []string
	}{
		{
			name:         "continue without new prompt",
			taskID:       "task-123",
			newPrompt:    "",
			originalTask: originalTask,
			expected: []string{
				"✓ Task continued successfully!",
				"task-123",
				"failed",
				"queued",
				"https://github.com/user/repo.git",
				"Fix the authentication bug",
				"2 → 3", // attempts increment
				"The task has been queued for retry",
			},
		},
		{
			name:         "continue with new prompt",
			taskID:       "task-123",
			newPrompt:    "Try a different approach",
			originalTask: originalTask,
			expected: []string{
				"✓ Task continued successfully!",
				"task-123",
				"failed",
				"queued",
				"Original Prompt:",
				"Fix the authentication bug",
				"New Prompt:",
				"Try a different approach",
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

			err := outputContinueTable(tt.taskID, tt.newPrompt, tt.originalTask)
			if err != nil {
				t.Fatalf("outputContinueTable failed: %v", err)
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

func TestOutputContinueJSON(t *testing.T) {
	taskID := "task-123"
	newPrompt := "Try a different approach"

	var buf bytes.Buffer
	
	// Temporarily redirect output
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	err := outputContinueJSON(taskID, newPrompt)
	if err != nil {
		t.Fatalf("outputContinueJSON failed: %v", err)
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
	if response["action"] != "continue" {
		t.Errorf("Expected action 'continue', got %v", response["action"])
	}
	if newPrompt != "" && response["new_prompt"] != newPrompt {
		t.Errorf("Expected new_prompt %s, got %v", newPrompt, response["new_prompt"])
	}
}

// Test prompt validation
func TestPromptValidation(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid prompt",
			prompt:  "Fix the authentication bug properly this time",
			wantErr: false,
		},
		{
			name:    "prompt too short",
			prompt:  "Fix",
			wantErr: true,
			errMsg:  "prompt must be between 10 and 1000 characters",
		},
		{
			name:    "prompt too long",
			prompt:  strings.Repeat("a", 1001),
			wantErr: true,
			errMsg:  "prompt must be between 10 and 1000 characters",
		},
		{
			name:    "dangerous prompt",
			prompt:  "Run rm -rf / to clean up the filesystem",
			wantErr: true,
			errMsg:  "prompt contains potentially dangerous content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll use the validation logic from start command since continue uses similar validation
			err := validateStartInputs("https://github.com/user/repo.git", tt.prompt)

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

// Test wait flag
func TestWaitFlag(t *testing.T) {
	cmd := NewContinueCommand()
	
	// Set wait flag
	cmd.Flags().Set("wait", "true")
	
	// Check that wait flag is properly set
	waitFlag, err := cmd.Flags().GetBool("wait")
	if err != nil {
		t.Fatalf("Failed to get wait flag: %v", err)
	}
	
	if !waitFlag {
		t.Error("Expected wait flag to be true")
	}
}

// Benchmark tests
func BenchmarkValidateContinuable(b *testing.B) {
	task := &TaskResponse{
		ID:     "task-123",
		Status: "failed",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateContinuable(task)
	}
}

func BenchmarkOutputContinueTable(b *testing.B) {
	now := time.Now()
	originalTask := &TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		Prompt:    "Fix the authentication bug",
		Status:    "failed",
		Attempts:  2,
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
		outputContinueTable("task-123", "", originalTask)
	}
}
