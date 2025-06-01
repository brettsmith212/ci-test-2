package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupTaskRoutes configures task-related routes
func SetupTaskRoutes(router *gin.RouterGroup) {
	// Task routes will be implemented in the next step
	// For now, just add a placeholder
	router.GET("/tasks", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Task routes coming soon",
			"status":  "placeholder",
		})
	})
}

// SetupHealthRoutes configures health check routes
func SetupHealthRoutes(router *gin.Engine) {
	router.GET("/health", HealthCheckHandler)
	router.GET("/health/ready", ReadinessCheckHandler)
	router.GET("/health/live", LivenessCheckHandler)
}

// SetupAPIRoutes configures all API routes
func SetupAPIRoutes(router *gin.Engine) {
	// Health routes
	SetupHealthRoutes(router)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Ping endpoint for basic connectivity test
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
				"version": "v1",
			})
		})

		// Task routes
		SetupTaskRoutes(v1)
	}
}
