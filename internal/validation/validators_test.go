package validation

import (
	"testing"
)

func TestValidateRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid github shorthand",
			repo:    "user/repo",
			wantErr: false,
		},
		{
			name:    "valid github https url",
			repo:    "https://github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid github https url without .git",
			repo:    "https://github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "valid gitlab url",
			repo:    "https://gitlab.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid bitbucket url",
			repo:    "https://bitbucket.org/user/repo.git",
			wantErr: false,
		},
		{
			name:     "empty repo",
			repo:     "",
			wantErr:  true,
			errorMsg: "repository URL cannot be empty",
		},
		{
			name:     "invalid shorthand format",
			repo:     "invalid",
			wantErr:  true,
			errorMsg: "repository must be in format 'owner/repo' or full Git URL",
		},
		{
			name:     "too many parts in shorthand",
			repo:     "user/repo/extra",
			wantErr:  true,
			errorMsg: "invalid repository format: must be 'owner/repo'",
		},
		{
			name:     "empty owner",
			repo:     "/repo",
			wantErr:  true,
			errorMsg: "repository owner and name cannot be empty",
		},
		{
			name:     "empty repo name",
			repo:     "user/",
			wantErr:  true,
			errorMsg: "repository owner and name cannot be empty",
		},
		{
			name:     "unsupported host",
			repo:     "https://example.com/user/repo.git",
			wantErr:  true,
			errorMsg: "unsupported repository host: example.com",
		},
		{
			name:     "invalid url format",
			repo:     "https://github.com/user",
			wantErr:  true,
			errorMsg: "invalid repository path: must be in format 'owner/repo'",
		},
		{
			name:     "repo url too long",
			repo:     "https://github.com/" + string(make([]byte, 500)) + "/repo",
			wantErr:  true,
			errorMsg: "repository URL too long (max 500 characters)",
		},
		{
			name:     "invalid characters in shorthand",
			repo:     "user$/repo",
			wantErr:  true,
			errorMsg: "repository owner and name can only contain letters, numbers, hyphens, underscores, and dots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryURL(tt.repo)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRepositoryURL() expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("ValidateRepositoryURL() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRepositoryURL() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidatePromptContent(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid prompt",
			prompt:  "Fix the authentication bug in the login handler",
			wantErr: false,
		},
		{
			name:    "valid long prompt",
			prompt:  "This is a longer prompt that describes in detail what needs to be fixed in the codebase, including specific files and functions that should be addressed.",
			wantErr: false,
		},
		{
			name:     "empty prompt",
			prompt:   "",
			wantErr:  true,
			errorMsg: "prompt cannot be empty",
		},
		{
			name:     "too short prompt",
			prompt:   "Fix bug",
			wantErr:  true,
			errorMsg: "prompt too short (minimum 10 characters)",
		},
		{
			name:     "prompt with only whitespace",
			prompt:   "   \n\t   ",
			wantErr:  true,
			errorMsg: "prompt too short (minimum 10 characters)",
		},
		{
			name:     "too long prompt",
			prompt:   string(make([]byte, 10001)),
			wantErr:  true,
			errorMsg: "prompt too long (maximum 10000 characters)",
		},
		{
			name:     "prompt with script tag",
			prompt:   "Fix this bug <script>alert('hack')</script>",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:     "prompt with javascript",
			prompt:   "Fix this bug javascript:alert('hack')",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:     "prompt with eval",
			prompt:   "Fix this bug and eval(malicious_code)",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:     "prompt with exec",
			prompt:   "Fix this bug exec(rm -rf /)",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:     "prompt with system call",
			prompt:   "Fix this bug system('rm -rf /')",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:     "prompt with rm -rf",
			prompt:   "Fix this bug rm -rf important_files",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:     "prompt with sudo",
			prompt:   "Fix this bug sudo rm important_files",
			wantErr:  true,
			errorMsg: "prompt contains potentially dangerous content",
		},
		{
			name:    "prompt with safe HTML",
			prompt:  "Fix the <button> element styling in the UI",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePromptContent(tt.prompt)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePromptContent() expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("ValidatePromptContent() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePromptContent() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidatePaginationParams(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		offset   int
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid params",
			limit:   50,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "valid params with offset",
			limit:   25,
			offset:  100,
			wantErr: false,
		},
		{
			name:    "zero limit",
			limit:   0,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "max limit",
			limit:   100,
			offset:  0,
			wantErr: false,
		},
		{
			name:     "negative limit",
			limit:    -1,
			offset:   0,
			wantErr:  true,
			errorMsg: "limit cannot be negative",
		},
		{
			name:     "limit too high",
			limit:    101,
			offset:   0,
			wantErr:  true,
			errorMsg: "limit cannot exceed 100",
		},
		{
			name:     "negative offset",
			limit:    50,
			offset:   -1,
			wantErr:  true,
			errorMsg: "offset cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaginationParams(tt.limit, tt.offset)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePaginationParams() expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("ValidatePaginationParams() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePaginationParams() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTranslateValidationErrors(t *testing.T) {
	// Test with nil error
	result := TranslateValidationErrors(nil)
	if len(result) != 0 {
		t.Errorf("TranslateValidationErrors(nil) expected empty slice, got %d errors", len(result))
	}

	// Test with non-validator error
	simpleErr := ValidationError{
		Field:   "test",
		Value:   "value",
		Tag:     "required",
		Message: "test is required",
	}
	
	if simpleErr.Field != "test" {
		t.Errorf("ValidationError.Field = %v, want %v", simpleErr.Field, "test")
	}
	
	if simpleErr.Message != "test is required" {
		t.Errorf("ValidationError.Message = %v, want %v", simpleErr.Message, "test is required")
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   string
	}{
		{
			name:   "empty errors",
			errors: ValidationErrors{},
			want:   "",
		},
		{
			name: "single error",
			errors: ValidationErrors{
				{Message: "field is required"},
			},
			want: "field is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Message: "field1 is required"},
				{Message: "field2 is invalid"},
			},
			want: "field1 is required; field2 is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errors.Error(); got != tt.want {
				t.Errorf("ValidationErrors.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
