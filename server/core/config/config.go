package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the configuration for the dashboard and capture server applications
type Config struct {
	WebAddr      string `json:"web_addr"`
	WebPort      int    `json:"web_port"`
	CapturePort  int    `json:"capture_port"`
	DatabasePath string `json:"database_path"`
	LogPath      string `json:"log_path"`
	LogLevel     string `json:"log_level"`
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() *Config {

	dbDir := "."

	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		dbDir = filepath.Join(homeDir, "cryospy")

		// Ensure the directory exists
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			dbDir = "."
		}
	}

	return &Config{
		WebAddr:      "127.0.0.1",
		WebPort:      8080,
		CapturePort:  8081,
		DatabasePath: filepath.Join(dbDir, "cryospy.db"),
		LogPath:      filepath.Join(dbDir, "logs"),
		LogLevel:     "info",
	}
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	// If no path is provided, use the default path in user's home directory
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, "cryospy", "config.json")
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, we can proceed with the default config
			return config, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.WebPort <= 0 || c.WebPort > 65535 {
		return fmt.Errorf("invalid web port: %d", c.WebPort)
	}
	if c.CapturePort <= 0 || c.CapturePort > 65535 {
		return fmt.Errorf("invalid capture port: %d", c.CapturePort)
	}
	return nil
}

// SaveConfig saves the configuration to a JSON file
func (c *Config) SaveConfig(path string) error {
	// If no path is provided, use the default path in user's home directory
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, "cryospy", "config.json")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}

	return nil
}
