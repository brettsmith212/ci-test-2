package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/brettsmith212/ci-test-2/internal/database"
)

// HealthResponse represents the structure of health check responses
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// HealthCheckHandler provides a basic health check endpoint
func HealthCheckHandler(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}

// ReadinessCheckHandler checks if the service is ready to accept traffic
func ReadinessCheckHandler(c *gin.Context) {
	checks := make(map[string]string)
	status := "ok"
	httpStatus := http.StatusOK

	// Check database connectivity
	if err := database.Health(); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		status = "not ready"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["database"] = "healthy"
	}

	// Add more checks as needed (Redis, external APIs, etc.)
	
	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	}

	c.JSON(httpStatus, response)
}

// LivenessCheckHandler checks if the service is alive (basic functionality)
func LivenessCheckHandler(c *gin.Context) {
	// Basic liveness check - if we can respond, we're alive
	response := HealthResponse{
		Status:    "alive",
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// DetailedHealthCheckHandler provides comprehensive health information
func DetailedHealthCheckHandler(c *gin.Context) {
	checks := make(map[string]string)
	status := "ok"
	httpStatus := http.StatusOK

	// Database health check
	if err := database.Health(); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		status = "degraded"
		if httpStatus == http.StatusOK {
			httpStatus = http.StatusServiceUnavailable
		}
	} else {
		checks["database"] = "healthy"
	}

	// Check database connection pool
	if db := database.GetDB(); db != nil {
		if sqlDB, err := db.DB(); err == nil {
			if stats := sqlDB.Stats(); stats.OpenConnections > 0 {
				checks["database_pool"] = "healthy"
			} else {
				checks["database_pool"] = "no connections"
				if status == "ok" {
					status = "degraded"
				}
			}
		}
	}

	// Memory usage check (basic)
	checks["memory"] = "healthy" // Placeholder for actual memory monitoring

	// Disk space check (placeholder)
	checks["disk"] = "healthy" // Placeholder for actual disk monitoring

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Checks:    checks,
	}

	c.JSON(httpStatus, response)
}
