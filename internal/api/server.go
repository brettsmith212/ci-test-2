package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/brettsmith212/ci-test-2/internal/config"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config) *Server {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	server := &Server{
		config: cfg,
		router: gin.New(),
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Custom logging middleware
	s.router.Use(LoggingMiddleware())

	// CORS middleware
	s.router.Use(CORSMiddleware())

	// Request ID middleware
	s.router.Use(RequestIDMiddleware())

	// Error handling middleware
	s.router.Use(ErrorHandlingMiddleware())
}

// setupRoutes configures all routes for the server
func (s *Server) setupRoutes() {
	// Health check routes
	s.router.GET("/health", HealthCheckHandler)
	s.router.GET("/health/ready", ReadinessCheckHandler)
	s.router.GET("/health/live", LivenessCheckHandler)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
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

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.config.Server.Address,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", s.config.Server.Address)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")

	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

// GetRouter returns the Gin router (useful for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *config.Config {
	return s.config
}
