package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ampx",
	Short: "CLI for CI-Driven Background Agent Orchestrator",
	Long: `ampx is a command-line interface for managing CI-driven Amp tasks.
It allows you to start, monitor, pause, resume, and manage background
tasks that integrate with your CI/CD pipeline.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().String("api-url", "http://localhost:8080", "Orchestrator API URL")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
}

// exitErrorf prints an error message and exits with code 1
func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
