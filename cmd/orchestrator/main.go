package main

import (
	"log"

	"github.com/brettsmith212/ci-test-2/internal/api"
	"github.com/brettsmith212/ci-test-2/internal/config"
	"github.com/brettsmith212/ci-test-2/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting CI-Driven Background Agent Orchestrator...")
	log.Printf("Server will listen on %s", cfg.Server.Address)
	log.Printf("Database path: %s", cfg.Database.Path)

	// Initialize database connection
	if err := database.Connect(cfg.Database.Path); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run database migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Test database health
	if err := database.Health(); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}

	log.Println("Database connected and migrations completed successfully")

	// Initialize Gin server with routes
	server := api.NewServer(cfg)
	
	log.Println("Orchestrator started successfully")

	// Start HTTP server
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
