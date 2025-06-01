package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

// LoggingMiddleware provides structured logging for HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[%s] %s %s %d %s %s\n",
				param.TimeStamp.Format("2006-01-02 15:04:05"),
				param.Method,
				param.Path,
				param.StatusCode,
				param.Latency,
				param.ClientIP,
			)
		},
		Output: log.Writer(),
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Allow localhost and common development origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://localhost:8081",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
			"http://127.0.0.1:8081",
		}

		// Check if origin is allowed
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is already provided
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate new ULID for request ID
			requestID = ulid.Make().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// ErrorHandlingMiddleware provides centralized error handling with improved error responses
func ErrorHandlingMiddleware() gin.HandlerFunc {
	errorHandler := GetErrorHandler()
	
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID := c.GetString("request_id")

			log.Printf("[ERROR] Request ID: %s, Error: %v", requestID, err.Err)

			// Don't override status if it's already set
			if c.Writer.Status() == 200 {
				errorHandler.HandleGenericError(c, err.Err)
			}
		}
	}
}

// ValidationMiddleware provides request validation with structured error responses
func ValidationMiddleware() gin.HandlerFunc {
	errorHandler := GetErrorHandler()
	
	return func(c *gin.Context) {
		// Store error handler in context for easy access
		c.Set("error_handler", errorHandler)
		c.Next()
	}
}

// ContentTypeValidationMiddleware validates content type for POST/PUT/PATCH requests
func ContentTypeValidationMiddleware() gin.HandlerFunc {
	errorHandler := GetErrorHandler()
	
	return func(c *gin.Context) {
		method := c.Request.Method
		
		// Only validate content type for requests that should have a body
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			
			// Check if content type is JSON
			if !strings.Contains(contentType, "application/json") {
				errorHandler.HandleBadRequestError(c, 
					"Invalid content type",
					"Content-Type must be application/json for this endpoint")
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// RequestSizeMiddleware limits request body size
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	errorHandler := GetErrorHandler()
	
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			errorHandler.HandleBadRequestError(c,
				"Request too large",
				fmt.Sprintf("Request body cannot exceed %d bytes", maxSize))
			c.Abort()
			return
		}
		
		// Limit the request body reader
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		
		c.Next()
	}
}

// HeaderValidationMiddleware validates required headers
func HeaderValidationMiddleware(requiredHeaders map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		missingHeaders := make(map[string]string)
		
		for header, description := range requiredHeaders {
			if c.GetHeader(header) == "" {
				missingHeaders[header] = fmt.Sprintf("%s header is required", description)
			}
		}
		
		if len(missingHeaders) > 0 {
			apiError := ValidationErrorFields(missingHeaders)
			apiError.RequestID = c.GetString("request_id")
			c.JSON(http.StatusBadRequest, apiError)
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// SecurityMiddleware adds basic security headers
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// RateLimitMiddleware provides basic rate limiting (placeholder for future implementation)
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement rate limiting logic
		// For now, just pass through
		c.Next()
	}
}
