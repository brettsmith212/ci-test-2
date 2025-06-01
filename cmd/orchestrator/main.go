package main

import (
	"log"
	"os"

	"github.com/brettsmith212/ci-test-2/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting CI-Driven Background Agent Orchestrator...")
	log.Printf("Server will listen on %s", cfg.Server.Address)

	// TODO: Initialize database connection
	// TODO: Initialize Gin server with routes
	// TODO: Start task dispatcher
	// TODO: Start HTTP server

	log.Println("Orchestrator started successfully")

	// Keep the process running
	select {}
}
