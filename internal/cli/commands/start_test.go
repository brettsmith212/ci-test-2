package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

func TestNewStartCommand(t *testing.T) {
	cmd := NewStartCommand()

	if cmd.Use != "start <repository> <prompt>" {
		t.Errorf("Expected use to be 'start <repository> <prompt>', got %s", cmd.Use)
	}

	if cmd.Short != "Start a new CI-driven Amp task" {
		t.Errorf("Expected short description to match, got %s", cmd.Short)
	}
}

func TestValidateStartInputs(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		prompt  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid github repo and prompt",
			repo:    "https://github.com/user/repo.git",
			prompt:  "Fix the bug",
			wantErr: false,
		},
		{
			name:    "valid gitlab repo",
			repo:    "https://gitlab.com/user/repo.git",
			prompt:  "Add new feature",
			wantErr: false,
		},
		{
			name:    "valid bitbucket repo",
			repo:    "https://bitbucket.org/user/repo.git",
			prompt:  "Update documentation",
			wantErr: false,
		},
		{
			name:    "valid ssh github repo",
			repo:    "git@github.com:user/repo.git",
			prompt:  "Fix tests",
			wantErr: false,
		},
		{
			name:    "empty repo",
			repo:    "",
			prompt:  "Fix the bug",
			wantErr: true,
			errMsg:  "repository URL cannot be empty",
		},
		{
			name:    "invalid repo prefix",
			repo:    "https://example.com/user/repo.git",
			prompt:  "Fix the bug",
			wantErr: true,
			errMsg:  "invalid repository URL",
		},
		{
			name:    "empty prompt",
			repo:    "https://github.com/user/repo.git",
			prompt:  "",
			wantErr: true,
			errMsg:  "prompt cannot be empty",
		},
		{
			name:    "prompt too short",
			repo:    "https://github.com/user/repo.git",
			prompt:  "Fix",
			wantErr: true,
			errMsg:  "prompt must be between 10 and 1000 characters",
		},
		{
			name:    "prompt too long",
			repo:    "https://github.com/user/repo.git",
			prompt:  strings.Repeat("a", 1001),
			wantErr: true,
			errMsg:  "prompt must be between 10 and 1000 characters",
		},
		{
			name:    "dangerous prompt with rm -rf",
			repo:    "https://github.com/user/repo.git",
			prompt:  "Run rm -rf / to clean up files",
			wantErr: true,
			errMsg:  "prompt contains potentially dangerous content: rm -rf",
		},
		{
			name:    "dangerous prompt with eval",
			repo:    "https://github.com/user/repo.git",
			prompt:  "Use eval() to execute this code dynamically",
			wantErr: true,
			errMsg:  "prompt contains potentially dangerous content: eval(",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStartInputs(tt.repo, tt.prompt)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
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

func TestStartCommandExecution(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		mockResponse   string
		mockStatusCode int
		outputFormat   string
		wantErr        bool
		errMsg         string
		checkOutput    func(t *testing.T, output string)
	}{
		{
			name:           "successful task creation - table format",
			args:           []string{"https://github.com/user/repo.git", "Fix the authentication bug"},
			mockResponse:   `{"id": "test-123", "branch": "amp/test"}`,
			mockStatusCode: 201,
			outputFormat:   "table",
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "✓ Task created successfully!") {
					t.Errorf("Expected success message in output")
				}
				if !strings.Contains(output, "test-123") {
					t.Errorf("Expected task ID in output")
				}
				if !strings.Contains(output, "amp/test") {
					t.Errorf("Expected branch in output")
				}
			},
		},
		{
			name:           "successful task creation - json format",
			args:           []string{"https://github.com/user/repo.git", "Fix the authentication bug"},
			mockResponse:   `{"id": "test-456", "branch": "amp/test2"}`,
			mockStatusCode: 201,
			outputFormat:   "json",
			wantErr:        false,
			checkOutput: func(t *testing.T, output string) {
				var response CreateTaskResponse
				if err := json.Unmarshal([]byte(output), &response); err != nil {
					t.Errorf("Expected valid JSON output, got error: %v", err)
				}
				if response.ID != "test-456" {
					t.Errorf("Expected ID test-456, got %s", response.ID)
				}
			},
		},
		{
			name:           "API error response",
			args:           []string{"https://github.com/user/repo.git", "Fix the authentication bug"},
			mockResponse:   `{"error": "Repository not accessible"}`,
			mockStatusCode: 400,
			outputFormat:   "table",
			wantErr:        true,
			errMsg:         "failed to create task",
		},
		{
			name:           "invalid repository URL",
			args:           []string{"invalid-repo", "Fix the authentication bug"},
			mockResponse:   "",
			mockStatusCode: 0,
			outputFormat:   "table",
			wantErr:        true,
			errMsg:         "invalid repository URL",
		},
		{
			name:           "prompt too short",
			args:           []string{"https://github.com/user/repo.git", "Fix"},
			mockResponse:   "",
			mockStatusCode: 0,
			outputFormat:   "table",
			wantErr:        true,
			errMsg:         "prompt must be between 10 and 1000 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server if we expect an API call
			var mockServer *httptest.Server
			if tt.mockStatusCode > 0 {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request method and path
					if r.Method != "POST" {
						t.Errorf("Expected POST request, got %s", r.Method)
					}
					if r.URL.Path != "/api/v1/tasks" {
						t.Errorf("Expected /api/v1/tasks path, got %s", r.URL.Path)
					}

					// Verify request body
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("Failed to read request body: %v", err)
					}

					var req CreateTaskRequest
					if err := json.Unmarshal(body, &req); err != nil {
						t.Errorf("Failed to parse request body: %v", err)
					}

					if req.Repo != tt.args[0] {
						t.Errorf("Expected repo %s, got %s", tt.args[0], req.Repo)
					}
					if req.Prompt != tt.args[1] {
						t.Errorf("Expected prompt %s, got %s", tt.args[1], req.Prompt)
					}

					w.WriteHeader(tt.mockStatusCode)
					w.Write([]byte(tt.mockResponse))
				}))
				defer mockServer.Close()
			}

			// Create command
			cmd := NewStartCommand()
			
			// Set output format flag
			if tt.outputFormat != "" {
				cmd.Flags().Set("output", tt.outputFormat)
			}

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set API URL if we have a mock server
			if mockServer != nil {
				cmd.Flags().Set("api-url", mockServer.URL)
			}

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

func TestOutputStartTable(t *testing.T) {
	tests := []struct {
		name     string
		response CreateTaskResponse
		repo     string
		prompt   string
		expected []string
	}{
		{
			name: "basic output",
			response: CreateTaskResponse{
				ID:     "task-123",
				Branch: "amp/branch",
			},
			repo:   "https://github.com/user/repo.git",
			prompt: "Fix the authentication bug",
			expected: []string{
				"✓ Task created successfully!",
				"task-123",
				"amp/branch",
				"https://github.com/user/repo.git",
				"Fix the authentication bug",
				"ampx logs task-123",
				"ampx list",
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

			err := outputStartTable(tt.response, tt.repo, tt.prompt)
			if err != nil {
				t.Fatalf("outputStartTable failed: %v", err)
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

func TestOutputJSON(t *testing.T) {
	response := CreateTaskResponse{
		ID:     "test-123",
		Branch: "amp/test",
	}

	var buf bytes.Buffer
	
	// Temporarily redirect output
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	err := outputJSON(response)
	if err != nil {
		t.Fatalf("outputJSON failed: %v", err)
	}

	output := buf.String()
	
	// Verify it's valid JSON
	var parsed CreateTaskResponse
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	if parsed.ID != response.ID {
		t.Errorf("Expected ID %s, got %s", response.ID, parsed.ID)
	}
	if parsed.Branch != response.Branch {
		t.Errorf("Expected Branch %s, got %s", response.Branch, parsed.Branch)
	}
}

// Benchmark tests
func BenchmarkValidateStartInputs(b *testing.B) {
	repo := "https://github.com/user/repo.git"
	prompt := "Fix the authentication bug in the login system"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateStartInputs(repo, prompt)
	}
}

func BenchmarkOutputStartTable(b *testing.B) {
	response := CreateTaskResponse{
		ID:     "task-123",
		Branch: "amp/branch",
	}
	repo := "https://github.com/user/repo.git"
	prompt := "Fix the authentication bug"

	var buf bytes.Buffer
	oldOutput := cli.GetOutput()
	cli.SetOutput(&buf)
	defer cli.SetOutput(oldOutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		outputStartTable(response, repo, prompt)
	}
}
