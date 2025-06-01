package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorType represents different types of application errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation_error"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeUnauthorized   ErrorType = "unauthorized"
	ErrorTypeForbidden      ErrorType = "forbidden"
	ErrorTypeRateLimit      ErrorType = "rate_limit_exceeded"
	ErrorTypeInternal       ErrorType = "internal_error"
	ErrorTypeBadRequest     ErrorType = "bad_request"
	ErrorTypeServiceUnavailable ErrorType = "service_unavailable"
)

// APIError represents a structured application error
type APIError struct {
	Type          ErrorType         `json:"error"`
	Message       string            `json:"message"`
	Details       string            `json:"details,omitempty"`
	Fields        map[string]string `json:"fields,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	Code          string            `json:"code,omitempty"`
	Documentation string            `json:"documentation,omitempty"`
}

// Error implements the error interface
func (e APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ErrorHandler provides centralized error handling for the API
type ErrorHandler struct{}

// NewErrorHandler creates a new ErrorHandler instance
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// HandleValidationError handles validation errors with detailed field information
func (h *ErrorHandler) HandleValidationError(c *gin.Context, err error) {
	requestID := c.GetString("request_id")
	
	// Check if it's our custom validation errors
	if validationErrs, ok := err.(ValidationErrors); ok {
		fields := make(map[string]string)
		var messages []string
		
		for _, validationErr := range validationErrs {
			fields[validationErr.Field] = validationErr.Message
			messages = append(messages, validationErr.Message)
		}
		
		apiError := APIError{
			Type:      ErrorTypeValidation,
			Message:   "Request validation failed",
			Details:   strings.Join(messages, "; "),
			Fields:    fields,
			RequestID: requestID,
		}
		
		c.JSON(http.StatusBadRequest, apiError)
		return
	}
	
	// Handle Gin binding errors
	validationErrs := TranslateValidationErrors(err)
	if len(validationErrs) > 0 {
		fields := make(map[string]string)
		var messages []string
		
		for _, validationErr := range validationErrs {
			fields[validationErr.Field] = validationErr.Message
			messages = append(messages, validationErr.Message)
		}
		
		apiError := APIError{
			Type:      ErrorTypeValidation,
			Message:   "Request validation failed",
			Details:   strings.Join(messages, "; "),
			Fields:    fields,
			RequestID: requestID,
		}
		
		c.JSON(http.StatusBadRequest, apiError)
		return
	}
	
	// Generic validation error
	apiError := APIError{
		Type:      ErrorTypeValidation,
		Message:   "Invalid request data",
		Details:   err.Error(),
		RequestID: requestID,
	}
	
	c.JSON(http.StatusBadRequest, apiError)
}

// HandleNotFoundError handles resource not found errors
func (h *ErrorHandler) HandleNotFoundError(c *gin.Context, resource string, identifier string) {
	requestID := c.GetString("request_id")
	
	apiError := APIError{
		Type:      ErrorTypeNotFound,
		Message:   fmt.Sprintf("%s not found", strings.Title(resource)),
		Details:   fmt.Sprintf("%s with identifier '%s' does not exist", resource, identifier),
		RequestID: requestID,
		Code:      "RESOURCE_NOT_FOUND",
	}
	
	c.JSON(http.StatusNotFound, apiError)
}

// HandleConflictError handles resource conflict errors
func (h *ErrorHandler) HandleConflictError(c *gin.Context, message string, details string) {
	requestID := c.GetString("request_id")
	
	apiError := APIError{
		Type:      ErrorTypeConflict,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Code:      "RESOURCE_CONFLICT",
	}
	
	c.JSON(http.StatusConflict, apiError)
}

// HandleUnauthorizedError handles authentication errors
func (h *ErrorHandler) HandleUnauthorizedError(c *gin.Context, message string) {
	requestID := c.GetString("request_id")
	
	if message == "" {
		message = "Authentication required"
	}
	
	apiError := APIError{
		Type:      ErrorTypeUnauthorized,
		Message:   message,
		RequestID: requestID,
		Code:      "AUTHENTICATION_REQUIRED",
	}
	
	c.JSON(http.StatusUnauthorized, apiError)
}

// HandleForbiddenError handles authorization errors
func (h *ErrorHandler) HandleForbiddenError(c *gin.Context, message string) {
	requestID := c.GetString("request_id")
	
	if message == "" {
		message = "Insufficient permissions"
	}
	
	apiError := APIError{
		Type:      ErrorTypeForbidden,
		Message:   message,
		RequestID: requestID,
		Code:      "INSUFFICIENT_PERMISSIONS",
	}
	
	c.JSON(http.StatusForbidden, apiError)
}

// HandleRateLimitError handles rate limiting errors
func (h *ErrorHandler) HandleRateLimitError(c *gin.Context, limit int, windowSeconds int) {
	requestID := c.GetString("request_id")
	
	apiError := APIError{
		Type:      ErrorTypeRateLimit,
		Message:   "Rate limit exceeded",
		Details:   fmt.Sprintf("Maximum %d requests per %d seconds exceeded", limit, windowSeconds),
		RequestID: requestID,
		Code:      "RATE_LIMIT_EXCEEDED",
	}
	
	// Add rate limit headers
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	c.Header("X-RateLimit-Window", fmt.Sprintf("%d", windowSeconds))
	
	c.JSON(http.StatusTooManyRequests, apiError)
}

// HandleInternalError handles internal server errors
func (h *ErrorHandler) HandleInternalError(c *gin.Context, message string, err error) {
	requestID := c.GetString("request_id")
	
	// Log the actual error for debugging (don't expose to client)
	if err != nil {
		c.Error(fmt.Errorf("internal error [%s]: %w", requestID, err))
	}
	
	if message == "" {
		message = "An internal error occurred"
	}
	
	apiError := APIError{
		Type:      ErrorTypeInternal,
		Message:   message,
		RequestID: requestID,
		Code:      "INTERNAL_SERVER_ERROR",
	}
	
	c.JSON(http.StatusInternalServerError, apiError)
}

// HandleBadRequestError handles bad request errors
func (h *ErrorHandler) HandleBadRequestError(c *gin.Context, message string, details string) {
	requestID := c.GetString("request_id")
	
	apiError := APIError{
		Type:      ErrorTypeBadRequest,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Code:      "BAD_REQUEST",
	}
	
	c.JSON(http.StatusBadRequest, apiError)
}

// HandleServiceUnavailableError handles service unavailable errors
func (h *ErrorHandler) HandleServiceUnavailableError(c *gin.Context, message string, retryAfterSeconds int) {
	requestID := c.GetString("request_id")
	
	if message == "" {
		message = "Service temporarily unavailable"
	}
	
	apiError := APIError{
		Type:      ErrorTypeServiceUnavailable,
		Message:   message,
		RequestID: requestID,
		Code:      "SERVICE_UNAVAILABLE",
	}
	
	if retryAfterSeconds > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
	}
	
	c.JSON(http.StatusServiceUnavailable, apiError)
}

// HandleGenericError handles errors based on common patterns
func (h *ErrorHandler) HandleGenericError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	
	errMsg := err.Error()
	
	// Pattern matching for common errors
	switch {
	case strings.Contains(errMsg, "not found"):
		h.HandleNotFoundError(c, "resource", "unknown")
	case strings.Contains(errMsg, "validation"):
		h.HandleValidationError(c, err)
	case strings.Contains(errMsg, "conflict") || strings.Contains(errMsg, "already exists"):
		h.HandleConflictError(c, "Resource conflict", errMsg)
	case strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "authentication"):
		h.HandleUnauthorizedError(c, errMsg)
	case strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "permission"):
		h.HandleForbiddenError(c, errMsg)
	case strings.Contains(errMsg, "rate limit"):
		h.HandleRateLimitError(c, 100, 3600) // Default rate limit
	default:
		h.HandleInternalError(c, "An error occurred", err)
	}
}

// ErrorResponse creates a simple error response (for backward compatibility)
func ErrorResponse(errorType ErrorType, message string) APIError {
	return APIError{
		Type:    errorType,
		Message: message,
	}
}

// ValidationErrorFields creates a validation error with field details
func ValidationErrorFields(fields map[string]string) APIError {
	var messages []string
	for _, msg := range fields {
		messages = append(messages, msg)
	}
	
	return APIError{
		Type:    ErrorTypeValidation,
		Message: "Validation failed",
		Details: strings.Join(messages, "; "),
		Fields:  fields,
	}
}

// GetErrorHandler returns a global error handler instance
var globalErrorHandler = NewErrorHandler()

func GetErrorHandler() *ErrorHandler {
	return globalErrorHandler
}
