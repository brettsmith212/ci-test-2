package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	if cmd.Use != "list" {
		t.Errorf("Expected use to be 'list', got %s", cmd.Use)
	}

	if cmd.Short != "List tasks" {
		t.Errorf("Expected short description to match, got %s", cmd.Short)
	}

	// Check flags exist
	flags := cmd.Flags()
	if flags.Lookup("status") == nil {
		t.Error("Expected --status flag to exist")
	}
	if flags.Lookup("limit") == nil {
		t.Error("Expected --limit flag to exist")
	}
	if flags.Lookup("offset") == nil {
		t.Error("Expected --offset flag to exist")
	}
	if flags.Lookup("output") == nil {
		t.Error("Expected --output flag to exist")
	}
	if flags.Lookup("watch") == nil {
		t.Error("Expected --watch flag to exist")
	}
	if flags.Lookup("repo") == nil {
		t.Error("Expected --repo flag to exist")
	}
}

func TestListCommandExecution(t *testing.T) {
	// Create sample task data
	now := time.Now()
	sampleTasks := []TaskResponse{
		{
			ID:        "task-123",
			Repo:      "https://github.com/user/repo1.git",
			Branch:    "amp/task1",
			ThreadID:  "thread-123",
			Prompt:    "Fix the authentication bug",
			Status:    "running",
			CIRunID:   nil,
			Attempts:  1,
			Summary:   "",
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-time.Minute * 30),
		},
		{
			ID:        "task-456",
			Repo:      "https://github.com/user/repo2.git",
			Branch:    "amp/task2",
			ThreadID:  "thread-456",
			Prompt:    "Add unit tests for user service",
			Status:    "queued",
			CIRunID:   nil,
			Attempts:  0,
			Summary:   "",
			CreatedAt: now.Add(-time.Hour * 2),
			UpdatedAt: now.Add(-time.Hour * 2),
		},
	}

	tests := []struct {
		name           string
		flags          map[string]string
		mockResponse   TaskListResponse
		mockStatusCode int
		wantErr        bool
		errMsg         string
		checkOutput    func(t *testing.T, output string)
		checkRequest   func(t *testing.T, r *http.Request)
	}{
		{
			name:  "successful list - table format",
			flags: map[string]string{"output": "table"},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks,
				Total: 2,
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "task-123") {
					t.Error("Expected task-123 in output")
				}
				if !strings.Contains(output, "task-456") {
					t.Error("Expected task-456 in output")
				}
				if !strings.Contains(output, "running") {
					t.Error("Expected running status in output")
				}
				if !strings.Contains(output, "queued") {
					t.Error("Expected queued status in output")
				}
			},
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/tasks" {
					t.Errorf("Expected /api/v1/tasks path, got %s", r.URL.Path)
				}
			},
		},
		{
			name:  "successful list - json format",
			flags: map[string]string{"output": "json"},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks[:1], // Only first task
				Total: 1,
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				var tasks []map[string]interface{}
				if err := json.Unmarshal([]byte(output), &tasks); err != nil {
					t.Errorf("Expected valid JSON output, got error: %v", err)
				}
				if len(tasks) != 1 {
					t.Errorf("Expected 1 task in JSON output, got %d", len(tasks))
				}
			},
		},
		{
			name:  "successful list - wide format",
			flags: map[string]string{"output": "wide"},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks,
				Total: 2,
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				// Wide format should include branch column
				if !strings.Contains(output, "BRANCH") {
					t.Error("Expected BRANCH column header in wide format")
				}
				if !strings.Contains(output, "amp/task1") {
					t.Error("Expected branch name in wide format")
				}
			},
		},
		{
			name: "list with status filter",
			flags: map[string]string{
				"status": "running",
				"output": "table",
			},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks[:1], // Only running task
				Total: 1,
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("status") != "running" {
					t.Errorf("Expected status=running query param, got %s", r.URL.Query().Get("status"))
				}
			},
		},
		{
			name: "list with pagination",
			flags: map[string]string{
				"limit":  "10",
				"offset": "5",
				"output": "table",
			},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks,
				Total: 15, // More than displayed
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Showing 2 of 15 tasks") {
					t.Error("Expected pagination info in output")
				}
			},
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("limit") != "10" {
					t.Errorf("Expected limit=10 query param, got %s", r.URL.Query().Get("limit"))
				}
				if r.URL.Query().Get("offset") != "5" {
					t.Errorf("Expected offset=5 query param, got %s", r.URL.Query().Get("offset"))
				}
			},
		},
		{
			name: "list with repo filter",
			flags: map[string]string{
				"repo":   "user/repo1",
				"output": "table",
			},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks[:1],
				Total: 1,
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("repo") != "user/repo1" {
					t.Errorf("Expected repo=user/repo1 query param, got %s", r.URL.Query().Get("repo"))
				}
			},
		},
		{
			name:  "empty task list",
			flags: map[string]string{"output": "table"},
			mockResponse: TaskListResponse{
				Tasks: []TaskResponse{},
				Total: 0,
			},
			mockStatusCode: 200,
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "No tasks found") {
					t.Error("Expected 'No tasks found' message in output")
				}
			},
		},
		{
			name:           "API error",
			flags:          map[string]string{"output": "table"},
			mockResponse:   TaskListResponse{},
			mockStatusCode: 500,
			wantErr:        true,
			errMsg:         "failed to list tasks",
		},
		{
			name:  "invalid output format",
			flags: map[string]string{"output": "xml"},
			mockResponse: TaskListResponse{
				Tasks: sampleTasks,
				Total: 2,
			},
			mockStatusCode: 200,
			wantErr:        true,
			errMsg:         "unsupported output format: xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkRequest != nil {
					tt.checkRequest(t, r)
				}

				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode == 200 {
					json.NewEncoder(w).Encode(tt.mockResponse)
				} else {
					w.Write([]byte(`{"error": "server error"}`))
				}
			}))
			defer mockServer.Close()

			// Create command
			cmd := NewListCommand()

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

func TestOutputTaskTable(t *testing.T) {
	now := time.Now()
	sampleTasks := []TaskResponse{
		{
			ID:        "task-123",
			Repo:      "https://github.com/user/repo.git",
			Branch:    "amp/task",
			Prompt:    "Fix the authentication bug",
			Status:    "running",
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-time.Minute * 30),
		},
	}

	tests := []struct {
		name     string
		response TaskListResponse
		expected []string
	}{
		{
			name: "single task",
			response: TaskListResponse{
				Tasks: sampleTasks,
				Total: 1,
			},
			expected: []string{
				"task-123",
				"running",
				"user/repo",
				"Fix the authentication bug",
			},
		},
		{
			name: "empty list",
			response: TaskListResponse{
				Tasks: []TaskResponse{},
				Total: 0,
			},
			expected: []string{
				"No tasks found",
			},
		},
		{
			name: "pagination info",
			response: TaskListResponse{
				Tasks: sampleTasks,
				Total: 10, // More than displayed
			},
			expected: []string{
				"task-123",
				"Showing 1 of 10 tasks",
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

			err := outputTaskTable(tt.response)
			if err != nil {
				t.Fatalf("outputTaskTable failed: %v", err)
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

func TestOutputTaskTableWide(t *testing.T) {
	now := time.Now()
	sampleTask := TaskResponse{
		ID:        "task-123",
		Repo:      "https://github.com/user/repo.git",
		Branch:    "amp/task",
		Prompt:    "Fix the authentication bug",
		Status:    "running",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now.Add(-time.Minute * 30),
	}

	response := TaskListResponse{
		Tasks: []TaskResponse{sampleTask},
		Total: 1,
	}

	var buf bytes.Buffer
	
	// Temporarily redirect output
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	err := outputTaskTableWide(response)
	if err != nil {
		t.Fatalf("outputTaskTableWide failed: %v", err)
	}

	output := buf.String()
	
	// Check for wide format specific columns
	expectedColumns := []string{"ID", "STATUS", "REPOSITORY", "BRANCH", "PROMPT", "CREATED", "UPDATED"}
	for _, col := range expectedColumns {
		if !strings.Contains(output, col) {
			t.Errorf("Expected column '%s' in wide format output", col)
		}
	}

	// Check for task data
	if !strings.Contains(output, "task-123") {
		t.Error("Expected task ID in output")
	}
	if !strings.Contains(output, "amp/task") {
		t.Error("Expected branch in output")
	}
}

// Test watch mode functionality (non-blocking test)
func TestWatchMode(t *testing.T) {
	// This is a simplified test since watch mode runs indefinitely
	// In a real scenario, you'd use context cancellation or timeout
	
	cmd := NewListCommand()
	
	// Set watch flag
	cmd.Flags().Set("watch", "true")
	
	// Check that watch flag is properly set
	watchFlag, err := cmd.Flags().GetBool("watch")
	if err != nil {
		t.Fatalf("Failed to get watch flag: %v", err)
	}
	
	if !watchFlag {
		t.Error("Expected watch flag to be true")
	}
}

// Benchmark tests
func BenchmarkOutputTaskTable(b *testing.B) {
	now := time.Now()
	sampleTasks := make([]TaskResponse, 100)
	for i := 0; i < 100; i++ {
		sampleTasks[i] = TaskResponse{
			ID:        fmt.Sprintf("task-%d", i),
			Repo:      "https://github.com/user/repo.git",
			Branch:    fmt.Sprintf("amp/task-%d", i),
			Prompt:    "Fix the authentication bug",
			Status:    "running",
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-time.Minute * 30),
		}
	}

	response := TaskListResponse{
		Tasks: sampleTasks,
		Total: 100,
	}

	var buf bytes.Buffer
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		outputTaskTable(response)
	}
}

func BenchmarkOutputTaskTableWide(b *testing.B) {
	now := time.Now()
	sampleTasks := make([]TaskResponse, 50)
	for i := 0; i < 50; i++ {
		sampleTasks[i] = TaskResponse{
			ID:        fmt.Sprintf("task-%d", i),
			Repo:      "https://github.com/user/repo.git",
			Branch:    fmt.Sprintf("amp/task-%d", i),
			Prompt:    "Fix the authentication bug",
			Status:    "running",
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-time.Minute * 30),
		}
	}

	response := TaskListResponse{
		Tasks: sampleTasks,
		Total: 50,
	}

	var buf bytes.Buffer
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		outputTaskTableWide(response)
	}
}
