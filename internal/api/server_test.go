package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/brettsmith212/ci-test-2/internal/config"
	"github.com/brettsmith212/ci-test-2/internal/database"
	"github.com/brettsmith212/ci-test-2/internal/models"
)

func setupTestDBForServer(t *testing.T) func() {
	// Create temporary test database
	tmpDir, err := os.MkdirTemp("", "server_test_*")
	require.NoError(t, err)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	// Initialize test database
	err = database.Connect(dbPath)
	require.NoError(t, err)
	
	// Run migrations
	err = database.GetDB().AutoMigrate(&models.Task{})
	require.NoError(t, err)
	
	// Return cleanup function
	return func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}
}

func setupTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Address: ":8080",
		},
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
	}
}

func TestNewServer(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	
	assert.NotNil(t, server)
	assert.NotNil(t, server.router)
	assert.Equal(t, cfg, server.config)
}

func TestServerHealthEndpoints(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		expectedKeys   []string
	}{
		{
			name:           "health_check",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			expectedKeys:   []string{"status", "timestamp"},
		},
		{
			name:           "readiness_check",
			endpoint:       "/health/ready",
			expectedStatus: http.StatusOK,
			expectedKeys:   []string{"status"},
		},
		{
			name:           "liveness_check",
			endpoint:       "/health/live",
			expectedStatus: http.StatusOK,
			expectedKeys:   []string{"status"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.endpoint, nil)
			require.NoError(t, err)
			
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			
			assert.Equal(t, tt.expectedStatus, resp.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(t, err)
			
			for _, key := range tt.expectedKeys {
				assert.Contains(t, response, key, "Response should contain key: %s", key)
			}
		})
	}
}

func TestServerPingEndpoint(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	req, err := http.NewRequest("GET", "/api/v1/ping", nil)
	require.NoError(t, err)
	
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusOK, resp.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "pong", response["message"])
	assert.Equal(t, "v1", response["version"])
}

func TestServerMiddlewareStack(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	// Test CORS headers
	req, err := http.NewRequest("OPTIONS", "/api/v1/ping", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Equal(t, "http://localhost:3000", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "POST")
	
	// Test security headers
	req, err = http.NewRequest("GET", "/api/v1/ping", nil)
	require.NoError(t, err)
	
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "nosniff", resp.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", resp.Header().Get("X-XSS-Protection"))
	
	// Test request ID middleware
	assert.NotEmpty(t, resp.Header().Get("X-Request-ID"))
}

func TestServerRequestSizeLimit(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	// Create a payload larger than 10MB
	largePayload := make([]byte, 11*1024*1024) // 11MB
	for i := range largePayload {
		largePayload[i] = 'a'
	}
	
	req, err := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(largePayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	
	var errorResp map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Equal(t, "bad_request", errorResp["error"])
}

func TestServerContentTypeValidation(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	tests := []struct {
		name           string
		method         string
		endpoint       string
		contentType    string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid_content_type",
			method:         "POST",
			endpoint:       "/api/v1/tasks",
			contentType:    "application/json",
			body:           `{"repo": "https://github.com/test/repo.git", "prompt": "Fix the authentication bug"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid_content_type",
			method:         "POST",
			endpoint:       "/api/v1/tasks",
			contentType:    "text/plain",
			body:           `{"repo": "https://github.com/test/repo.git", "prompt": "Fix the authentication bug"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad_request",
		},
		{
			name:           "missing_content_type",
			method:         "POST",
			endpoint:       "/api/v1/tasks",
			contentType:    "",
			body:           `{"repo": "https://github.com/test/repo.git", "prompt": "Fix the authentication bug"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad_request",
		},
		{
			name:           "get_request_no_content_type_required",
			method:         "GET",
			endpoint:       "/api/v1/ping",
			contentType:    "",
			body:           "",
			expectedStatus: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.endpoint, bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			
			assert.Equal(t, tt.expectedStatus, resp.Code)
			
			if tt.expectedError != "" {
				var errorResp map[string]interface{}
				err = json.Unmarshal(resp.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResp["error"])
			}
		})
	}
}

func TestServerErrorHandling(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	// Test 404 error handling
	req, err := http.NewRequest("GET", "/non-existent-endpoint", nil)
	require.NoError(t, err)
	
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusNotFound, resp.Code)
	
	// 404 responses from Gin don't return JSON by default, they return HTML
	assert.Contains(t, resp.Body.String(), "404")
}

func TestServerJSONResponseStructure(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	req, err := http.NewRequest("GET", "/api/v1/ping", nil)
	require.NoError(t, err)
	
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header().Get("Content-Type"))
	
	// Verify JSON is valid
	var jsonResp map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &jsonResp)
	require.NoError(t, err)
}

func TestServerGracefulShutdown(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	cfg.Server.Address = ":0" // Use random available port
	server := NewServer(cfg)
	
	// Test stopping server before starting
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	err := server.Stop(ctx)
	assert.NoError(t, err)
}

func TestServerConfigAccess(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	
	// Test getting config
	returnedConfig := server.GetConfig()
	assert.Equal(t, cfg, returnedConfig)
	assert.Equal(t, ":8080", returnedConfig.Server.Address)
	
	// Test getting router
	router := server.GetRouter()
	assert.NotNil(t, router)
	assert.IsType(t, &gin.Engine{}, router)
}

func TestServerTaskEndpointsIntegration(t *testing.T) {
	cleanup := setupTestDBForServer(t)
	defer cleanup()
	
	gin.SetMode(gin.TestMode)
	
	cfg := setupTestConfig()
	server := NewServer(cfg)
	router := server.GetRouter()
	
	// Test full workflow: create -> get -> list -> update
	
	// 1. Create task
	createPayload := map[string]string{
		"repo":   "https://github.com/test/repo.git",
		"prompt": "Fix the authentication bug in the system",
	}
	body, _ := json.Marshal(createPayload)
	
	createReq, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	
	assert.Equal(t, http.StatusCreated, createResp.Code)
	
	var createResult map[string]interface{}
	err := json.Unmarshal(createResp.Body.Bytes(), &createResult)
	require.NoError(t, err)
	
	taskID := createResult["id"].(string)
	assert.NotEmpty(t, taskID)
	
	// 2. Get task
	getReq, _ := http.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	
	assert.Equal(t, http.StatusOK, getResp.Code)
	
	// 3. List tasks
	listReq, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	
	assert.Equal(t, http.StatusOK, listResp.Code)
	
	var listResult map[string]interface{}
	err = json.Unmarshal(listResp.Body.Bytes(), &listResult)
	require.NoError(t, err)
	
	tasks := listResult["tasks"].([]interface{})
	assert.Len(t, tasks, 1)
	
	// 4. Update task
	updatePayload := map[string]string{
		"action": "abort",
	}
	updateBody, _ := json.Marshal(updatePayload)
	
	updateReq, _ := http.NewRequest("PATCH", "/api/v1/tasks/"+taskID, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)
	
	assert.Equal(t, http.StatusNoContent, updateResp.Code)
	
	// 5. Get active tasks
	activeReq, _ := http.NewRequest("GET", "/api/v1/tasks/active", nil)
	activeResp := httptest.NewRecorder()
	router.ServeHTTP(activeResp, activeReq)
	
	assert.Equal(t, http.StatusOK, activeResp.Code)
}
