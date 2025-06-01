package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name              string
		existingRequestID string
		wantRequestID     bool
	}{
		{
			name:          "generates new request ID when none provided",
			wantRequestID: true,
		},
		{
			name:              "uses existing request ID when provided",
			existingRequestID: "existing-request-id",
			wantRequestID:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(RequestIDMiddleware())
			r.GET("/test", func(c *gin.Context) {
				requestID := c.GetString("request_id")
				if tt.wantRequestID && requestID == "" {
					t.Error("Expected request_id to be set in context")
				}
				if tt.existingRequestID != "" && requestID != tt.existingRequestID {
					t.Errorf("Expected request_id = %v, got %v", tt.existingRequestID, requestID)
				}
				c.JSON(200, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingRequestID != "" {
				req.Header.Set("X-Request-ID", tt.existingRequestID)
			}

			c.Request = req
			r.ServeHTTP(w, req)

			// Check response header
			if tt.wantRequestID {
				responseRequestID := w.Header().Get("X-Request-ID")
				if responseRequestID == "" {
					t.Error("Expected X-Request-ID header in response")
				}
				if tt.existingRequestID != "" && responseRequestID != tt.existingRequestID {
					t.Errorf("Expected X-Request-ID header = %v, got %v", tt.existingRequestID, responseRequestID)
				}
			}
		})
	}
}

func TestContentTypeValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		method         string
		contentType    string
		expectedStatus int
		shouldAbort    bool
	}{
		{
			name:           "GET request - no validation",
			method:         "GET",
			contentType:    "",
			expectedStatus: 200,
			shouldAbort:    false,
		},
		{
			name:           "POST with valid content type",
			method:         "POST",
			contentType:    "application/json",
			expectedStatus: 200,
			shouldAbort:    false,
		},
		{
			name:           "POST with charset in content type",
			method:         "POST",
			contentType:    "application/json; charset=utf-8",
			expectedStatus: 200,
			shouldAbort:    false,
		},
		{
			name:           "POST with invalid content type",
			method:         "POST",
			contentType:    "text/plain",
			expectedStatus: 400,
			shouldAbort:    true,
		},
		{
			name:           "PUT with valid content type",
			method:         "PUT",
			contentType:    "application/json",
			expectedStatus: 200,
			shouldAbort:    false,
		},
		{
			name:           "PATCH with valid content type",
			method:         "PATCH",
			contentType:    "application/json",
			expectedStatus: 200,
			shouldAbort:    false,
		},
		{
			name:           "POST without content type",
			method:         "POST",
			contentType:    "",
			expectedStatus: 400,
			shouldAbort:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(RequestIDMiddleware()) // Add this for error handler
			r.Use(ContentTypeValidationMiddleware())
			r.Any("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			body := `{"test": "data"}`
			req := httptest.NewRequest(tt.method, "/test", strings.NewReader(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			c.Request = req
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRequestSizeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	maxSize := int64(100) // 100 bytes for testing
	
	tests := []struct {
		name           string
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "small request body",
			bodySize:       50,
			expectedStatus: 200,
		},
		{
			name:           "exact max size",
			bodySize:       100,
			expectedStatus: 200,
		},
		{
			name:           "oversized request body",
			bodySize:       150,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(RequestIDMiddleware()) // Add this for error handler
			r.Use(RequestSizeMiddleware(maxSize))
			r.POST("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			body := strings.Repeat("a", tt.bodySize)
			req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.ContentLength = int64(tt.bodySize)

			c.Request = req
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHeaderValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	requiredHeaders := map[string]string{
		"Authorization": "Authorization",
		"X-API-Key":     "API Key",
	}
	
	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name: "all required headers present",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-API-Key":     "key123",
			},
			expectedStatus: 200,
		},
		{
			name: "missing authorization header",
			headers: map[string]string{
				"X-API-Key": "key123",
			},
			expectedStatus: 400,
		},
		{
			name: "missing api key header",
			headers: map[string]string{
				"Authorization": "Bearer token123",
			},
			expectedStatus: 400,
		},
		{
			name:           "missing all headers",
			headers:        map[string]string{},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(RequestIDMiddleware()) // Add this for error context
			r.Use(HeaderValidationMiddleware(requiredHeaders))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			c.Request = req
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		expectCORS     bool
	}{
		{
			name:           "allowed origin localhost:3000",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name:           "allowed origin 127.0.0.1:8080",
			origin:         "http://127.0.0.1:8080",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     true,
		},
		{
			name:           "disallowed origin",
			origin:         "http://evil.com",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     false,
		},
		{
			name:           "OPTIONS preflight request",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedStatus: 204,
			expectCORS:     true,
		},
		{
			name:           "no origin header",
			origin:         "",
			method:         "GET",
			expectedStatus: 200,
			expectCORS:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(CORSMiddleware())
			r.Any("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			c.Request = req
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check CORS headers
			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS && corsHeader != tt.origin {
				t.Errorf("Expected CORS header %v, got %v", tt.origin, corsHeader)
			}
			if !tt.expectCORS && corsHeader != "" {
				t.Errorf("Expected no CORS header, got %v", corsHeader)
			}

			// Check other CORS headers are always present
			if w.Header().Get("Access-Control-Allow-Methods") == "" {
				t.Error("Expected Access-Control-Allow-Methods header")
			}
			if w.Header().Get("Access-Control-Allow-Headers") == "" {
				t.Error("Expected Access-Control-Allow-Headers header")
			}
		})
	}
}

func TestSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(SecurityMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":   "nosniff",
		"X-Frame-Options":          "DENY",
		"X-XSS-Protection":         "1; mode=block",
		"Referrer-Policy":          "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected %s header = %v, got %v", header, expectedValue, actualValue)
		}
	}
}

func TestErrorHandlingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		shouldError    bool
		expectedStatus int
	}{
		{
			name:           "no error",
			shouldError:    false,
			expectedStatus: 200,
		},
		{
			name:           "with error",
			shouldError:    true,
			expectedStatus: 500, // Should be handled by error middleware
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(RequestIDMiddleware()) // Needed for error context
			r.Use(ErrorHandlingMiddleware())
			r.GET("/test", func(c *gin.Context) {
				if tt.shouldError {
					c.Error(NewValidationError("test error"))
					return
				}
				c.JSON(200, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(ValidationMiddleware())
	r.GET("/test", func(c *gin.Context) {
		// Check if error handler is set in context
		errorHandler := c.MustGet("error_handler")
		if errorHandler == nil {
			t.Error("Expected error_handler to be set in context")
		}
		c.JSON(200, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
