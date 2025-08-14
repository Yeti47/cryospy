package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	ServerURL           string `json:"server_url"`
	CameraDevice        string `json:"camera_device"`
	BufferSize          int    `json:"buffer_size"`           // Number of clips to buffer in memory
	SettingsSyncSeconds int    `json:"settings_sync_seconds"` // How often to sync settings from server (in seconds)

	// Video processing configuration
	VideoCodec        string  `json:"video_codec"`         // Video codec for processing (e.g., "mpeg4", "libopenh264")
	VideoOutputFormat string  `json:"video_output_format"` // Output container format (e.g., "mp4", "avi")
	VideoBitRate      string  `json:"video_bitrate"`       // Video bitrate for compression (e.g., "500k", "1M")
	CaptureCodec      string  `json:"capture_codec"`       // Codec for initial capture (e.g., "MJPG", "MP4V")
	CaptureFrameRate  float64 `json:"capture_framerate"`   // Frame rate for video capture (e.g., 15.0, 30.0)
	MotionSensitivity float64 `json:"motion_sensitivity"`  // Motion detection sensitivity as percentage (e.g., 1.0 = 1% of pixels)
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, create a default one
			defaultConfig := &Config{
				ClientID:            "your-client-id",
				ClientSecret:        "your-client-secret",
				ServerURL:           "http://localhost:8080",
				CameraDevice:        "/dev/video0",
				BufferSize:          3,
				SettingsSyncSeconds: 300, // 5 minutes default

				// Video processing defaults
				VideoCodec:        "libopenh264", // Open-source H.264 codec that works on Fedora
				VideoOutputFormat: "mp4",         // Standard MP4 container
				VideoBitRate:      "500k",        // 500 Kbps for reasonable file sizes
				CaptureCodec:      "MJPG",        // Motion JPEG for reliable capture
				CaptureFrameRate:  15.0,          // 15 FPS for smaller files
				MotionSensitivity: 10.0,          // 10% of pixels changing for motion detection
			}

			if err := saveConfig(filename, defaultConfig); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}

			return nil, fmt.Errorf("config file '%s' was created with default values. Please edit it with your settings and run the application again", filename)
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

	// Set video processing defaults
	if config.VideoCodec == "" {
		config.VideoCodec = "mpeg4"
	}
	if config.VideoOutputFormat == "" {
		config.VideoOutputFormat = "mp4"
	}
	if config.VideoBitRate == "" {
		config.VideoBitRate = "500k"
	}
	if config.CaptureCodec == "" {
		config.CaptureCodec = "MJPG"
	}
	if config.CaptureFrameRate == 0 {
		config.CaptureFrameRate = 15.0
	}
	if config.MotionSensitivity == 0 {
		config.MotionSensitivity = 10.0
	}

	return &config, nil
}

// ConfigOverrides holds potential override values for configuration
type ConfigOverrides struct {
	ClientID            *string
	ClientSecret        *string
	ServerURL           *string
	CameraDevice        *string
	BufferSize          *int
	SettingsSyncSeconds *int
	VideoCodec          *string
	VideoOutputFormat   *string
	VideoBitRate        *string
	CaptureCodec        *string
	CaptureFrameRate    *float64
	MotionSensitivity   *float64
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

	// Video processing parameter overrides
	if overrides.VideoCodec != nil && *overrides.VideoCodec != "" {
		c.VideoCodec = *overrides.VideoCodec
	}
	if overrides.VideoOutputFormat != nil && *overrides.VideoOutputFormat != "" {
		c.VideoOutputFormat = *overrides.VideoOutputFormat
	}
	if overrides.VideoBitRate != nil && *overrides.VideoBitRate != "" {
		c.VideoBitRate = *overrides.VideoBitRate
	}
	if overrides.CaptureCodec != nil && *overrides.CaptureCodec != "" {
		c.CaptureCodec = *overrides.CaptureCodec
	}
	if overrides.CaptureFrameRate != nil && *overrides.CaptureFrameRate > 0 {
		c.CaptureFrameRate = *overrides.CaptureFrameRate
	}
	if overrides.MotionSensitivity != nil && *overrides.MotionSensitivity > 0 {
		c.MotionSensitivity = *overrides.MotionSensitivity
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
