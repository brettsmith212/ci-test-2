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
tasks that integrate with your CI/CD pipeline.

Examples:
  ampx start https://github.com/user/repo.git "Fix the bug"
  ampx list --status=running
  ampx logs <task-id>
  ampx abort <task-id>`,
	Version: "1.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help
		cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().String("api-url", "http://localhost:8080", "Orchestrator API URL")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	
	// Add subcommands
	addConfigCommand()
	addPingCommand()
}

// addConfigCommand adds the config subcommand
func addConfigCommand() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  "Show and manage CLI configuration settings",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := LoadConfig(cmd)
			if err != nil {
				exitErrorf("Failed to load config: %v", err)
			}
			
			if err := config.ValidateConfig(); err != nil {
				exitErrorf("Invalid config: %v", err)
			}
			
			fmt.Printf("Configuration:\n")
			fmt.Printf("  API URL: %s\n", config.APIUrl)
			fmt.Printf("  Verbose: %v\n", config.Verbose)
			
			if ConfigExists() {
				configPath, _ := GetConfigPath()
				fmt.Printf("  Config file: %s\n", configPath)
			} else {
				fmt.Printf("  Config file: Not found (using defaults)\n")
			}
		},
	}
	
	rootCmd.AddCommand(configCmd)
}

// addPingCommand adds the ping subcommand
func addPingCommand() {
	pingCmd := &cobra.Command{
		Use:   "ping",
		Short: "Test connection to the API server",
		Long:  "Test connectivity to the orchestrator API server",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := LoadConfig(cmd)
			if err != nil {
				exitErrorf("Failed to load config: %v", err)
			}
			
			client := NewClient(config)
			
			fmt.Printf("Pinging API server at %s...\n", config.APIUrl)
			
			if err := client.Ping(); err != nil {
				exitErrorf("Ping failed: %v", err)
			}
			
			fmt.Println("API server is reachable!")
			
			// Also check health
			if err := client.CheckHealth(); err != nil {
				fmt.Printf("Warning: Health check failed: %v\n", err)
			} else {
				fmt.Println("API server is healthy!")
			}
		},
	}
	
	rootCmd.AddCommand(pingCmd)
}

// exitErrorf prints an error message and exits with code 1
func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
