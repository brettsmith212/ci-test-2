package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/brettsmith212/ci-test-2/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting CI-Driven Background Agent Worker...")

	// TODO: Initialize database connection
	// TODO: Initialize GitHub client
	// TODO: Initialize Amp client
	// TODO: Start worker loop

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, gracefully stopping worker...")
		cancel()
	}()

	log.Println("Worker started successfully")

	// TODO: Run worker with context
	<-ctx.Done()
	log.Println("Worker stopped")
}
