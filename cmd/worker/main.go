package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/database"
	"github.com/brettsmith212/ci-test-2/internal/services"
	"github.com/brettsmith212/ci-test-2/internal/worker"
	"github.com/spf13/cobra"
)

var (
	dbPath         string
	workDir        string
	ampPath        string
	githubToken    string
	pollInterval   time.Duration
	maxConcurrency int
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "worker",
		Short: "Amp task worker for processing background tasks",
		Long:  `The worker processes tasks from the orchestrator by cloning repositories, running Amp prompts, and creating pull requests.`,
		Run:   runWorker,
	}

	// Define flags
	rootCmd.Flags().StringVar(&dbPath, "db", "./orchestrator.db", "Path to the SQLite database")
	rootCmd.Flags().StringVar(&workDir, "work-dir", "./work", "Working directory for repository operations")
	rootCmd.Flags().StringVar(&ampPath, "amp-path", "", "Path to Amp CLI binary (default: search in PATH)")
	rootCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub token for API access (can also use GITHUB_TOKEN env var)")
	rootCmd.Flags().DurationVar(&pollInterval, "poll-interval", 10*time.Second, "Interval for polling new tasks")
	rootCmd.Flags().IntVar(&maxConcurrency, "max-concurrency", 3, "Maximum number of concurrent tasks")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runWorker(cmd *cobra.Command, args []string) {
	log.Println("Starting Amp worker...")

	// Check for GitHub token in environment if not provided via flag
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	// Create absolute path for work directory
	workDirAbs, err := filepath.Abs(workDir)
	if err != nil {
		log.Fatalf("Failed to resolve work directory: %v", err)
	}

	// Initialize database connection
	log.Printf("Connecting to database: %s", dbPath)
	if err := database.Connect(dbPath); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize task service
	taskSvc := services.NewTaskServiceDefault()

	// Create worker configuration
	config := &worker.Config{
		PollInterval:   pollInterval,
		MaxConcurrency: maxConcurrency,
		WorkDir:        workDirAbs,
		AmpPath:        ampPath,
		GitHubToken:    githubToken,
		DatabasePath:   dbPath,
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create and start worker
	w := worker.New(config, taskSvc)

	// Set up graceful shutdown

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		log.Println("Initiating graceful shutdown...")
		w.Stop()
	}()

	// Start worker
	log.Printf("Worker configuration:")
	log.Printf("  Poll interval: %v", config.PollInterval)
	log.Printf("  Max concurrency: %d", config.MaxConcurrency)
	log.Printf("  Work directory: %s", config.WorkDir)
	log.Printf("  Amp path: %s", config.AmpPath)
	log.Printf("  GitHub token: %s", maskToken(config.GitHubToken))

	if err := w.Start(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}

	log.Println("Worker stopped")
}

func validateConfig(config *worker.Config) error {
	// Check if Amp is available
	ampOps := worker.NewAmpOperations(config.AmpPath)
	if err := ampOps.CheckInstallation(); err != nil {
		log.Printf("Warning: Amp CLI check failed: %v", err)
		log.Println("Worker will continue but may fail when processing tasks")
	} else {
		log.Println("Amp CLI installation verified")
	}

	// Validate work directory
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return err
	}

	log.Printf("Work directory ready: %s", config.WorkDir)
	return nil
}

func maskToken(token string) string {
	if token == "" {
		return "<not set>"
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}
