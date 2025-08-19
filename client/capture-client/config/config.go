package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	ClientID             string `json:"client_id"`
	ClientSecret         string `json:"client_secret"`
	ServerURL            string `json:"server_url"`
	CameraDevice         string `json:"camera_device"`
	BufferSize           int    `json:"buffer_size"`            // Number of clips to buffer in memory
	SettingsSyncSeconds  int    `json:"settings_sync_seconds"`  // How often to sync settings from server (in seconds)
	ServerTimeoutSeconds int    `json:"server_timeout_seconds"` // HTTP timeout for server requests (in seconds)
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, create a default one
			defaultConfig := &Config{
				ClientID:             "your-client-id",
				ClientSecret:         "your-client-secret",
				ServerURL:            "http://localhost:8080",
				CameraDevice:         "/dev/video0",
				BufferSize:           3,
				SettingsSyncSeconds:  300, // 5 minutes default
				ServerTimeoutSeconds: 30,  // 30 seconds default
			}
			if err := saveConfig(filename, defaultConfig); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}
			fmt.Printf("Default config file created at %s\n", filename)
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults for missing values
	if config.CameraDevice == "" {
		config.CameraDevice = "/dev/video0"
	}
	if config.BufferSize == 0 {
		config.BufferSize = 3
	}
	if config.SettingsSyncSeconds == 0 {
		config.SettingsSyncSeconds = 300 // 5 minutes default
	}
	if config.ServerTimeoutSeconds == 0 {
		config.ServerTimeoutSeconds = 30 // 30 seconds default
	}

	return &config, nil
}

// ConfigOverrides holds potential override values for configuration
type ConfigOverrides struct {
	ClientID             *string
	ClientSecret         *string
	ServerURL            *string
	CameraDevice         *string
	BufferSize           *int
	SettingsSyncSeconds  *int
	ServerTimeoutSeconds *int
}

// Override allows overriding specific configuration values using ConfigOverrides struct
func (c *Config) Override(overrides ConfigOverrides) {
	if overrides.ClientID != nil && *overrides.ClientID != "" {
		c.ClientID = *overrides.ClientID
	}
	if overrides.ClientSecret != nil && *overrides.ClientSecret != "" {
		c.ClientSecret = *overrides.ClientSecret
	}
	if overrides.ServerURL != nil && *overrides.ServerURL != "" {
		c.ServerURL = *overrides.ServerURL
	}
	if overrides.CameraDevice != nil && *overrides.CameraDevice != "" {
		c.CameraDevice = *overrides.CameraDevice
	}
	if overrides.BufferSize != nil && *overrides.BufferSize > 0 {
		c.BufferSize = *overrides.BufferSize
	}
	if overrides.SettingsSyncSeconds != nil && *overrides.SettingsSyncSeconds > 0 {
		c.SettingsSyncSeconds = *overrides.SettingsSyncSeconds
	}
	if overrides.ServerTimeoutSeconds != nil && *overrides.ServerTimeoutSeconds > 0 {
		c.ServerTimeoutSeconds = *overrides.ServerTimeoutSeconds
	}
}

// saveConfig saves a configuration to a JSON file
func saveConfig(filename string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
