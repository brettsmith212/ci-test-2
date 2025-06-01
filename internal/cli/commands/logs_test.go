package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

func TestNewLogsCommand(t *testing.T) {
	cmd := NewLogsCommand()

	if cmd.Use != "logs <task-id>" {
		t.Errorf("Expected use to be 'logs <task-id>', got %s", cmd.Use)
	}

	if cmd.Short != "Show logs for a task" {
		t.Errorf("Expected short description to match, got %s", cmd.Short)
	}

	// Check flags exist
	flags := cmd.Flags()
	if flags.Lookup("follow") == nil {
		t.Error("Expected --follow flag to exist")
	}
	if flags.Lookup("tail") == nil {
		t.Error("Expected --tail flag to exist")
	}
	if flags.Lookup("output") == nil {
		t.Error("Expected --output flag to exist")
	}
}

func TestLogsCommandExecution(t *testing.T) {
	now := time.Now()
	sampleTask := TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		ThreadID:  "thread-123",
		Prompt:    "Fix the authentication bug",
		Status:    "running",
		CIRunID:   nil,
		Attempts:  1,
		Summary:   "Working on authentication fixes",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Minute * 30),
	}

	tests := []struct {
		name           string
		args           []string
		flags          map[string]string
		mockResponse   TaskResponse
		mockStatusCode int
		wantErr        bool
		errMsg         string
		checkOutput    func(t *testing.T, output string)
		checkRequest   func(t *testing.T, r *http.Request)
	}{
		{
			name:           "successful logs display - table format",
			args:           []string{"task-123"},
			flags:          map[string]string{"output": "table"},
			mockResponse:   sampleTask,
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Task task-123") {
					t.Error("Expected task ID in header")
				}
				if !strings.Contains(output, "running") {
					t.Error("Expected status in output")
				}
				if !strings.Contains(output, "user/repo") {
					t.Error("Expected repository in output")
				}
				if !strings.Contains(output, "Fix the authentication bug") {
					t.Error("Expected prompt in output")
				}
				if !strings.Contains(output, "Working on authentication fixes") {
					t.Error("Expected summary in output")
				}
			},
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/tasks/task-123" {
					t.Errorf("Expected /api/v1/tasks/task-123 path, got %s", r.URL.Path)
				}
			},
		},
		{
			name:           "successful logs display - json format",
			args:           []string{"task-456"},
			flags:          map[string]string{"output": "json"},
			mockResponse:   sampleTask,
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				var task map[string]interface{}
				if err := json.Unmarshal([]byte(output), &task); err != nil {
					t.Errorf("Expected valid JSON output, got error: %v", err)
				}
				if task["id"] != sampleTask.ID {
					t.Errorf("Expected ID %s in JSON, got %v", sampleTask.ID, task["id"])
				}
				if task["status"] != sampleTask.Status {
					t.Errorf("Expected status %s in JSON, got %v", sampleTask.Status, task["status"])
				}
			},
		},
		{
			name:           "task not found",
			args:           []string{"nonexistent"},
			flags:          map[string]string{"output": "table"},
			mockResponse:   TaskResponse{},
			mockStatusCode: 404,
			wantErr:        true,
			errMsg:         "failed to get task",
		},
		{
			name:           "missing task ID argument",
			args:           []string{},
			flags:          map[string]string{"output": "table"},
			mockResponse:   TaskResponse{},
			mockStatusCode: 0,
			wantErr:        true,
			errMsg:         "accepts 1 arg(s), received 0",
		},
		{
			name:           "invalid output format",
			args:           []string{"task-123"},
			flags:          map[string]string{"output": "xml"},
			mockResponse:   sampleTask,
			mockStatusCode: 200,
			wantErr:        true,
			errMsg:         "unsupported output format: xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server if we expect an API call
			var mockServer *httptest.Server
			if tt.mockStatusCode > 0 {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.checkRequest != nil {
						tt.checkRequest(t, r)
					}

					w.WriteHeader(tt.mockStatusCode)
					if tt.mockStatusCode == 200 {
						json.NewEncoder(w).Encode(tt.mockResponse)
					} else {
						w.Write([]byte(`{"error": "task not found"}`))
					}
				}))
				defer mockServer.Close()
			}

			// Create command
			cmd := NewLogsCommand()

			// Set flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}

			// Set API URL if we have a mock server
			if mockServer != nil {
				cmd.Flags().Set("api-url", mockServer.URL)
			}

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

				// Check output
				if tt.checkOutput != nil {
					tt.checkOutput(t, buf.String())
				}
			}
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"queued", false},
		{"running", false},
		{"retrying", false},
		{"needs_review", false},
		{"success", true},
		{"failed", true},
		{"error", true},
		{"aborted", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := isTerminalStatus(tt.status)
			if result != tt.expected {
				t.Errorf("Expected isTerminalStatus(%s) = %v, got %v", tt.status, tt.expected, result)
			}
		})
	}
}

func TestOutputTaskUpdate(t *testing.T) {
	now := time.Now()
	task := TaskResponse{
		ID:       "task-123",
		Status:   "running",
		Summary:  "Processing authentication fixes",
		CIRunID:  intPtr(12345),
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name       string
		task       TaskResponse
		lastStatus string
		expected   []string
	}{
		{
			name:       "initial status update",
			task:       task,
			lastStatus: "",
			expected: []string{
				"Task task-123:",
				"running",
				"Processing authentication fixes",
				"CI Run: 12345",
			},
		},
		{
			name:       "status transition",
			task:       task,
			lastStatus: "queued",
			expected: []string{
				"Task task-123:",
				"queued → running",
				"Processing authentication fixes",
				"CI Run: 12345",
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

			outputTaskUpdate(tt.task, tt.lastStatus)

			output := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
				}
			}
		})
	}
}

// Test follow mode behavior (simplified since it runs indefinitely)
func TestFollowFlag(t *testing.T) {
	cmd := NewLogsCommand()
	
	// Set follow flag
	cmd.Flags().Set("follow", "true")
	
	// Check that follow flag is properly set
	followFlag, err := cmd.Flags().GetBool("follow")
	if err != nil {
		t.Fatalf("Failed to get follow flag: %v", err)
	}
	
	if !followFlag {
		t.Error("Expected follow flag to be true")
	}
}

func TestTailFlag(t *testing.T) {
	cmd := NewLogsCommand()
	
	// Set tail flag
	cmd.Flags().Set("tail", "100")
	
	// Check that tail flag is properly set
	tailFlag, err := cmd.Flags().GetInt("tail")
	if err != nil {
		t.Fatalf("Failed to get tail flag: %v", err)
	}
	
	if tailFlag != 100 {
		t.Errorf("Expected tail flag to be 100, got %d", tailFlag)
	}
}

// Test the old outputTaskLogs function if it still exists
func TestOutputTaskLogs(t *testing.T) {
	now := time.Now()
	task := TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		ThreadID:  "thread-123",
		Prompt:    "Fix the authentication bug",
		Status:    "running",
		CIRunID:   intPtr(12345),
		Attempts:  2,
		Summary:   "Working on authentication fixes",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Minute * 30),
	}

	var buf bytes.Buffer
	
	// Temporarily redirect output
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	err := outputTaskLogs(task)
	if err != nil {
		t.Fatalf("outputTaskLogs failed: %v", err)
	}

	output := buf.String()
	
	// Check for expected content
	expectedContent := []string{
		"Task Details: task-123",
		"Status:",
		"running",
		"Repository:",
		"https://github.com/user/repo.git",
		"Branch:",
		"amp/task",
		"Thread ID:",
		"thread-123",
		"Attempts:",
		"CI Run ID:",
		"12345",
		"Prompt:",
		"Fix the authentication bug",
		"Summary:",
		"Working on authentication fixes",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
		}
	}
}

// Test status-specific information display
func TestStatusSpecificInfo(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		status   string
		expected []string
	}{
		{
			name:   "queued status",
			status: "queued",
			expected: []string{
				"Task is waiting to be processed by a worker",
			},
		},
		{
			name:   "running status",
			status: "running",
			expected: []string{
				"Task is currently being processed by Amp",
			},
		},
		{
			name:   "retrying status",
			status: "retrying",
			expected: []string{
				"Task failed and is being retried",
			},
		},
		{
			name:   "needs_review status",
			status: "needs_review",
			expected: []string{
				"Task requires manual review before proceeding",
			},
		},
		{
			name:   "success status",
			status: "success",
			expected: []string{
				"Task has completed successfully",
			},
		},
		{
			name:   "failed status",
			status: "failed",
			expected: []string{
				"Task has failed",
			},
		},
		{
			name:   "aborted status",
			status: "aborted",
			expected: []string{
				"Task was manually aborted",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := TaskResponse{
				ID:        "task-123",
				Repo:      "https://github.com/user/repo.git",
				Status:    tt.status,
				Prompt:    "Fix the authentication bug",
				CreatedAt: now,
				UpdatedAt: now,
			}

			var buf bytes.Buffer
			
			// Temporarily redirect output
			oldOutput := cli.GetOutput()
			cli.SetOutput(&buf)
			defer cli.SetOutput(oldOutput)

			err := outputTaskLogs(task)
			if err != nil {
				t.Fatalf("outputTaskLogs failed: %v", err)
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

// Helper function to create int pointer
func intPtr(i int64) *int64 {
	return &i
}

// Benchmark tests
func BenchmarkOutputTaskLogs(b *testing.B) {
	now := time.Now()
	task := TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		ThreadID:  "thread-123",
		Prompt:    "Fix the authentication bug",
		Status:    "running",
		CIRunID:   intPtr(12345),
		Attempts:  2,
		Summary:   "Working on authentication fixes",
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
		outputTaskLogs(task)
	}
}

func BenchmarkIsTerminalStatus(b *testing.B) {
	statuses := []string{"queued", "running", "success", "failed", "aborted"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		isTerminalStatus(status)
	}
}
