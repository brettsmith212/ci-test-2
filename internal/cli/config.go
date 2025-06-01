package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config represents the CLI configuration
type Config struct {
	APIUrl  string `json:"api_url" mapstructure:"api_url"`
	Verbose bool   `json:"verbose" mapstructure:"verbose"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		APIUrl:  "http://localhost:8080",
		Verbose: false,
	}
}

// LoadConfig loads configuration from file, environment variables, and command flags
func LoadConfig(cmd *cobra.Command) (*Config, error) {
	config := DefaultConfig()

	// Set up viper
	viper.SetConfigName("ampx")
	viper.SetConfigType("json")

	// Add config paths
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "ampx"))
		viper.AddConfigPath(home)
	}
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("AMPX")
	viper.AutomaticEnv()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is okay, we'll use defaults
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with command line flags if they are set
	if cmd.Flag("api-url").Changed {
		apiURL, err := cmd.Flags().GetString("api-url")
		if err != nil {
			return nil, fmt.Errorf("failed to get api-url flag: %w", err)
		}
		config.APIUrl = apiURL
	}

	if cmd.Flag("verbose").Changed {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return nil, fmt.Errorf("failed to get verbose flag: %w", err)
		}
		config.Verbose = verbose
	}

	return config, nil
}

// SaveConfig saves the current configuration to the config file
func (c *Config) SaveConfig() error {
	// Determine config directory
	var configDir string
	if home, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(home, ".config", "ampx")
	} else {
		configDir = "."
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Config file path
	configFile := filepath.Join(configDir, "ampx.json")

	// Marshal config to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	var configDir string
	if home, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(home, ".config", "ampx")
	} else {
		configDir = "."
	}

	return filepath.Join(configDir, "ampx.json"), nil
}

// ConfigExists checks if a config file exists
func ConfigExists() bool {
	configPath, err := GetConfigPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(configPath)
	return err == nil
}

// ValidateConfig validates the configuration values
func (c *Config) ValidateConfig() error {
	if c.APIUrl == "" {
		return fmt.Errorf("api_url cannot be empty")
	}

	// Basic URL validation
	if c.APIUrl[:4] != "http" && c.APIUrl[:5] != "https" {
		return fmt.Errorf("api_url must start with http:// or https://")
	}

	return nil
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("APIUrl: %s, Verbose: %v", c.APIUrl, c.Verbose)
}

// GetAPIUrl returns the API URL with proper formatting
func (c *Config) GetAPIUrl() string {
	url := c.APIUrl
	// Remove trailing slash if present
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	return url
}

// GetAPIEndpoint returns the full API endpoint URL for a given path
func (c *Config) GetAPIEndpoint(path string) string {
	baseURL := c.GetAPIUrl()
	if path[0] != '/' {
		path = "/" + path
	}
	return baseURL + path
}
