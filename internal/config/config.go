package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	GitHub   GitHubConfig
	Amp      AmpConfig
	Worker   WorkerConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Address string
	Port    int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string
}

// GitHubConfig holds GitHub integration configuration
type GitHubConfig struct {
	AppID          string
	PrivateKeyPath string
	Token          string
}

// AmpConfig holds Amp CLI configuration
type AmpConfig struct {
	Command string
	Timeout int // seconds
}

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	MaxRetries      int
	RetryDelay      int // seconds
	PollInterval    int // seconds
	ConcurrentTasks int
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Address: getEnv("SERVER_ADDRESS", "localhost:8080"),
			Port:    getEnvAsInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_PATH", "orchestrator.db"),
		},
		GitHub: GitHubConfig{
			AppID:          getEnv("GITHUB_APP_ID", ""),
			PrivateKeyPath: getEnv("GITHUB_PRIVATE_KEY_PATH", ""),
			Token:          getEnv("GITHUB_TOKEN", ""),
		},
		Amp: AmpConfig{
			Command: getEnv("AMP_COMMAND", "amp"),
			Timeout: getEnvAsInt("AMP_TIMEOUT", 300), // 5 minutes
		},
		Worker: WorkerConfig{
			MaxRetries:      getEnvAsInt("WORKER_MAX_RETRIES", 3),
			RetryDelay:      getEnvAsInt("WORKER_RETRY_DELAY", 60),
			PollInterval:    getEnvAsInt("WORKER_POLL_INTERVAL", 30),
			ConcurrentTasks: getEnvAsInt("WORKER_CONCURRENT_TASKS", 1),
		},
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
