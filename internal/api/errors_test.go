package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError APIError
		want     string
	}{
		{
			name: "error with details",
			apiError: APIError{
				Type:    ErrorTypeValidation,
				Message: "Validation failed",
				Details: "Field is required",
			},
			want: "validation_error: Validation failed (Field is required)",
		},
		{
			name: "error without details",
			apiError: APIError{
				Type:    ErrorTypeNotFound,
				Message: "Resource not found",
			},
			want: "not_found: Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiError.Error(); got != tt.want {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorHandler_HandleValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "validation error",
			err:            ValidationErrors{{Field: "repo", Message: "repo is required"}},
			expectedStatus: http.StatusBadRequest,
			expectedType:   "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("request_id", "test-request-id")

			handler.HandleValidationError(c, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandleValidationError() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			// Check if response was written
			if w.Body.Len() == 0 {
				t.Error("HandleValidationError() did not write response body")
			}
		})
	}
}

func TestErrorHandler_HandleNotFoundError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	handler.HandleNotFoundError(c, "task", "123")

	if w.Code != http.StatusNotFound {
		t.Errorf("HandleNotFoundError() status = %v, want %v", w.Code, http.StatusNotFound)
	}

	if w.Body.Len() == 0 {
		t.Error("HandleNotFoundError() did not write response body")
	}
}

func TestErrorHandler_HandleConflictError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	handler.HandleConflictError(c, "Resource already exists", "Task with this name already exists")

	if w.Code != http.StatusConflict {
		t.Errorf("HandleConflictError() status = %v, want %v", w.Code, http.StatusConflict)
	}

	if w.Body.Len() == 0 {
		t.Error("HandleConflictError() did not write response body")
	}
}

func TestErrorHandler_HandleUnauthorizedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with custom message",
			message: "Invalid token",
		},
		{
			name:    "with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("request_id", "test-request-id")

			handler.HandleUnauthorizedError(c, tt.message)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("HandleUnauthorizedError() status = %v, want %v", w.Code, http.StatusUnauthorized)
			}

			if w.Body.Len() == 0 {
				t.Error("HandleUnauthorizedError() did not write response body")
			}
		})
	}
}

func TestErrorHandler_HandleForbiddenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	handler.HandleForbiddenError(c, "Access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("HandleForbiddenError() status = %v, want %v", w.Code, http.StatusForbidden)
	}

	if w.Body.Len() == 0 {
		t.Error("HandleForbiddenError() did not write response body")
	}
}

func TestErrorHandler_HandleRateLimitError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	handler.HandleRateLimitError(c, 100, 3600)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("HandleRateLimitError() status = %v, want %v", w.Code, http.StatusTooManyRequests)
	}

	if w.Body.Len() == 0 {
		t.Error("HandleRateLimitError() did not write response body")
	}

	// Check rate limit headers
	if w.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("X-RateLimit-Limit header = %v, want 100", w.Header().Get("X-RateLimit-Limit"))
	}

	if w.Header().Get("X-RateLimit-Window") != "3600" {
		t.Errorf("X-RateLimit-Window header = %v, want 3600", w.Header().Get("X-RateLimit-Window"))
	}
}

func TestErrorHandler_HandleInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	tests := []struct {
		name    string
		message string
		err     error
	}{
		{
			name:    "with error and message",
			message: "Database connection failed",
			err:     NewValidationError("test error"),
		},
		{
			name:    "with empty message",
			message: "",
			err:     NewValidationError("test error"),
		},
		{
			name:    "with nil error",
			message: "Something went wrong",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("request_id", "test-request-id")

			handler.HandleInternalError(c, tt.message, tt.err)

			if w.Code != http.StatusInternalServerError {
				t.Errorf("HandleInternalError() status = %v, want %v", w.Code, http.StatusInternalServerError)
			}

			if w.Body.Len() == 0 {
				t.Error("HandleInternalError() did not write response body")
			}
		})
	}
}

func TestErrorHandler_HandleBadRequestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	handler.HandleBadRequestError(c, "Invalid request", "Missing required field")

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleBadRequestError() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	if w.Body.Len() == 0 {
		t.Error("HandleBadRequestError() did not write response body")
	}
}

func TestErrorHandler_HandleServiceUnavailableError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewErrorHandler()
	
	tests := []struct {
		name             string
		message          string
		retryAfterSeconds int
	}{
		{
			name:              "with retry after",
			message:           "Service temporarily unavailable",
			retryAfterSeconds: 60,
		},
		{
			name:              "without retry after",
			message:           "Service down",
			retryAfterSeconds: 0,
		},
		{
			name:              "with empty message",
			message:           "",
			retryAfterSeconds: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("request_id", "test-request-id")

			handler.HandleServiceUnavailableError(c, tt.message, tt.retryAfterSeconds)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("HandleServiceUnavailableError() status = %v, want %v", w.Code, http.StatusServiceUnavailable)
			}

			if w.Body.Len() == 0 {
				t.Error("HandleServiceUnavailableError() did not write response body")
			}

			if tt.retryAfterSeconds > 0 {
				expectedHeader := "60"
				if tt.retryAfterSeconds == 30 {
					expectedHeader = "30"
				}
				if w.Header().Get("Retry-After") != expectedHeader {
					t.Errorf("Retry-After header = %v, want %v", w.Header().Get("Retry-After"), expectedHeader)
				}
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	result := ErrorResponse(ErrorTypeValidation, "Test message")
	
	if result.Type != ErrorTypeValidation {
		t.Errorf("ErrorResponse().Type = %v, want %v", result.Type, ErrorTypeValidation)
	}
	
	if result.Message != "Test message" {
		t.Errorf("ErrorResponse().Message = %v, want %v", result.Message, "Test message")
	}
}

func TestValidationErrorFields(t *testing.T) {
	fields := map[string]string{
		"field1": "error1",
		"field2": "error2",
	}
	
	result := ValidationErrorFields(fields)
	
	if result.Type != ErrorTypeValidation {
		t.Errorf("ValidationErrorFields().Type = %v, want %v", result.Type, ErrorTypeValidation)
	}
	
	if result.Message != "Validation failed" {
		t.Errorf("ValidationErrorFields().Message = %v, want %v", result.Message, "Validation failed")
	}
	
	if len(result.Fields) != 2 {
		t.Errorf("ValidationErrorFields().Fields length = %v, want 2", len(result.Fields))
	}
}

// Helper function to create a validation error for testing
func NewValidationError(message string) error {
	return ValidationErrors{{Message: message}}
}
