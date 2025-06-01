package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/brettsmith212/ci-test-2/internal/api/handlers"
)

// SetupTaskRoutes configures task-related routes
func SetupTaskRoutes(router *gin.RouterGroup) {
	taskHandler := handlers.NewTaskHandler()

	// Task CRUD routes
	router.POST("/tasks", taskHandler.CreateTask)
	router.GET("/tasks", taskHandler.ListTasks)
	router.GET("/tasks/:id", taskHandler.GetTask)
	router.PATCH("/tasks/:id", taskHandler.UpdateTask)

	// Additional task routes
	router.GET("/tasks/active", taskHandler.GetActiveTasks)
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
