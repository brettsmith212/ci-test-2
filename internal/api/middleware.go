package api

import (
	"fmt"
	"log"

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

// ErrorHandlingMiddleware provides centralized error handling
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID := c.GetString("request_id")

			log.Printf("[ERROR] Request ID: %s, Error: %v", requestID, err.Err)

			// Don't override status if it's already set
			if c.Writer.Status() == 200 {
				c.JSON(500, gin.H{
					"error":      "Internal server error",
					"request_id": requestID,
				})
			}
		}
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
