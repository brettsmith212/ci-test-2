package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// RegisterCustomValidators registers custom validation rules with Gin's validator
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("git_repo", validateGitRepo)
		v.RegisterValidation("task_prompt", validateTaskPrompt)
		v.RegisterValidation("task_status", validateTaskStatus)
		v.RegisterValidation("task_action", validateTaskAction)
	}
}

// validateGitRepo validates Git repository URLs and formats
func validateGitRepo(fl validator.FieldLevel) bool {
	repo := fl.Field().String()
	
	if repo == "" {
		return false
	}

	// Check for basic patterns
	// 1. Full Git URLs (https://github.com/user/repo.git)
	// 2. GitHub shorthand (user/repo)
	// 3. GitHub URLs without .git suffix (https://github.com/user/repo)
	
	// Pattern 1: Full Git URLs
	gitURLPattern := regexp.MustCompile(`^https?://[a-zA-Z0-9\-\.]+/[a-zA-Z0-9\-_\.]+/[a-zA-Z0-9\-_\.]+(?:\.git)?/?$`)
	if gitURLPattern.MatchString(repo) {
		return true
	}
	
	// Pattern 2: GitHub shorthand (user/repo)
	shorthandPattern := regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+/[a-zA-Z0-9\-_\.]+$`)
	if shorthandPattern.MatchString(repo) {
		return true
	}
	
	return false
}

// validateTaskPrompt validates task prompt content
func validateTaskPrompt(fl validator.FieldLevel) bool {
	prompt := fl.Field().String()
	
	if prompt == "" {
		return false
	}
	
	// Check length constraints
	if len(prompt) < 10 {
		return false
	}
	
	if len(prompt) > 10000 {
		return false
	}
	
	// Check for malicious content patterns
	maliciousPatterns := []string{
		"<script",
		"javascript:",
		"eval(",
		"exec(",
		"system(",
	}
	
	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return false
		}
	}
	
	return true
}

// validateTaskStatus validates task status values
func validateTaskStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	
	validStatuses := []string{
		"queued",
		"running", 
		"retrying",
		"needs_review",
		"success",
		"failed",
		"aborted",
	}
	
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	
	return false
}

// validateTaskAction validates task action values
func validateTaskAction(fl validator.FieldLevel) bool {
	action := fl.Field().String()
	
	validActions := []string{
		"continue",
		"abort",
	}
	
	for _, validAction := range validActions {
		if action == validAction {
			return true
		}
	}
	
	return false
}

// ValidateRepositoryURL performs comprehensive repository URL validation
func ValidateRepositoryURL(repo string) error {
	if repo == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}
	
	// Check length
	if len(repo) > 500 {
		return fmt.Errorf("repository URL too long (max 500 characters)")
	}
	
	// Try to parse as URL if it looks like a full URL
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		parsedURL, err := url.Parse(repo)
		if err != nil {
			return fmt.Errorf("invalid repository URL format")
		}
		
		// Check for supported hosts
		supportedHosts := []string{
			"github.com",
			"gitlab.com", 
			"bitbucket.org",
		}
		
		isSupported := false
		for _, host := range supportedHosts {
			if parsedURL.Host == host {
				isSupported = true
				break
			}
		}
		
		if !isSupported {
			return fmt.Errorf("unsupported repository host: %s", parsedURL.Host)
		}
		
		// Validate path structure
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) < 2 {
			return fmt.Errorf("invalid repository path: must be in format 'owner/repo'")
		}
	} else {
		// Validate shorthand format (owner/repo)
		if !strings.Contains(repo, "/") {
			return fmt.Errorf("repository must be in format 'owner/repo' or full Git URL")
		}
		
		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format: must be 'owner/repo'")
		}
		
		// Validate owner and repo names
		for _, part := range parts {
			if part == "" {
				return fmt.Errorf("repository owner and name cannot be empty")
			}
			
			// Check for valid characters
			if !regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`).MatchString(part) {
				return fmt.Errorf("repository owner and name can only contain letters, numbers, hyphens, underscores, and dots")
			}
		}
	}
	
	return nil
}

// ValidatePromptContent performs comprehensive prompt validation
func ValidatePromptContent(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	
	if len(strings.TrimSpace(prompt)) < 10 {
		return fmt.Errorf("prompt too short (minimum 10 characters)")
	}
	
	if len(prompt) > 10000 {
		return fmt.Errorf("prompt too long (maximum 10000 characters)")
	}
	
	// Check for potentially malicious content
	maliciousPatterns := []string{
		"<script",
		"javascript:",
		"eval(",
		"exec(",
		"system(",
		"rm -rf",
		"sudo ",
	}
	
	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return fmt.Errorf("prompt contains potentially dangerous content")
		}
	}
	
	return nil
}

// ValidatePaginationParams validates pagination parameters
func ValidatePaginationParams(limit, offset int) error {
	if limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	
	if limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}
	
	if offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	
	return nil
}

// TranslateValidationErrors converts validator errors to our custom format
func TranslateValidationErrors(err error) ValidationErrors {
	var validationErrors ValidationErrors
	
	if validatorErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validatorErrs {
			validationError := ValidationError{
				Field: fieldErr.Field(),
				Value: fmt.Sprintf("%v", fieldErr.Value()),
				Tag:   fieldErr.Tag(),
			}
			
			// Generate human-readable messages
			switch fieldErr.Tag() {
			case "required":
				validationError.Message = fmt.Sprintf("%s is required", fieldErr.Field())
			case "min":
				validationError.Message = fmt.Sprintf("%s must be at least %s characters", fieldErr.Field(), fieldErr.Param())
			case "max":
				validationError.Message = fmt.Sprintf("%s cannot exceed %s characters", fieldErr.Field(), fieldErr.Param())
			case "email":
				validationError.Message = fmt.Sprintf("%s must be a valid email address", fieldErr.Field())
			case "oneof":
				validationError.Message = fmt.Sprintf("%s must be one of: %s", fieldErr.Field(), fieldErr.Param())
			case "git_repo":
				validationError.Message = fmt.Sprintf("%s must be a valid Git repository URL or 'owner/repo' format", fieldErr.Field())
			case "task_prompt":
				validationError.Message = fmt.Sprintf("%s must be between 10-10000 characters and contain no malicious content", fieldErr.Field())
			case "task_status":
				validationError.Message = fmt.Sprintf("%s must be a valid task status", fieldErr.Field())
			case "task_action":
				validationError.Message = fmt.Sprintf("%s must be either 'continue' or 'abort'", fieldErr.Field())
			default:
				validationError.Message = fmt.Sprintf("%s failed validation: %s", fieldErr.Field(), fieldErr.Tag())
			}
			
			validationErrors = append(validationErrors, validationError)
		}
	}
	
	return validationErrors
}
