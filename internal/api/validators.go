package api

import (
	"github.com/brettsmith212/ci-test-2/internal/validation"
)

// Re-export validation types and functions for backward compatibility
type ValidationError = validation.ValidationError
type ValidationErrors = validation.ValidationErrors

// RegisterCustomValidators registers custom validation rules with Gin's validator
func RegisterCustomValidators() {
	validation.RegisterCustomValidators()
}

// ValidateRepositoryURL performs comprehensive repository URL validation
func ValidateRepositoryURL(repo string) error {
	return validation.ValidateRepositoryURL(repo)
}

// ValidatePromptContent performs comprehensive prompt validation
func ValidatePromptContent(prompt string) error {
	return validation.ValidatePromptContent(prompt)
}

// ValidatePaginationParams validates pagination parameters
func ValidatePaginationParams(limit, offset int) error {
	return validation.ValidatePaginationParams(limit, offset)
}

// TranslateValidationErrors converts validator errors to our custom format
func TranslateValidationErrors(err error) ValidationErrors {
	return validation.TranslateValidationErrors(err)
}
