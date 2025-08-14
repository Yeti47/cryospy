package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/models"
)

// MockCaptureServerClient is a mock implementation for testing
type MockCaptureServerClient struct {
	settings  *models.ClientSettings
	uploads   []UploadRecord
	outputDir string
}

// UploadRecord tracks uploaded clips for testing
type UploadRecord struct {
	Timestamp time.Time
	Size      int
	MimeType  string
	FilePath  string // Path where the clip was saved
}

// NewMockCaptureServerClient creates a new mock client for testing
func NewMockCaptureServerClient() *MockCaptureServerClient {
	// Create default test settings
	defaultSettings := &models.ClientSettings{
		ID:                    "test-client",
		StorageLimitMegabytes: 1000,
		ClipDurationSeconds:   30,
		MotionOnly:            true,
		Grayscale:             false,
		DownscaleResolution:   "720p",
	}

	// Create output directory for saved clips
	outputDir := "./mock-uploads"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("[MOCK] Warning: Failed to create output directory %s: %v", outputDir, err)
	} else {
		log.Printf("[MOCK] Created output directory: %s", outputDir)
	}

	return &MockCaptureServerClient{
		settings:  defaultSettings,
		uploads:   make([]UploadRecord, 0),
		outputDir: outputDir,
	}
}

// GetClientSettings returns mock client settings
func (m *MockCaptureServerClient) GetClientSettings(ctx context.Context) (*models.ClientSettings, error) {
	log.Println("[MOCK] Fetching client settings...")

	// Simulate some network delay
	time.Sleep(100 * time.Millisecond)

	// Return a copy of the settings
	settingsCopy := *m.settings
	log.Printf("[MOCK] Returning settings: MotionOnly=%v, Duration=%ds, Resolution=%s",
		settingsCopy.MotionOnly,
		settingsCopy.ClipDurationSeconds,
		settingsCopy.DownscaleResolution)

	return &settingsCopy, nil
}

// UploadClip simulates uploading a video clip
func (m *MockCaptureServerClient) UploadClip(ctx context.Context, videoData []byte, mimeType string, duration time.Duration, hasMotion bool) error {
	log.Printf("[MOCK] Uploading clip: %d bytes, type: %s, duration: %v, motion: %v", len(videoData), mimeType, duration, hasMotion)

	// Generate filename using server's clip title logic
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05")

	// Use the actual duration passed in
	durationStr := fmt.Sprintf("%.0f", duration.Seconds())

	// Simulate motion detection result (use the actual hasMotion value)
	motionStr := "nomotion"
	if hasMotion {
		motionStr = "motion"
	}

	// Use mp4 extension (matching the mimeType)
	extension := "mp4"

	// Create title in server format: {timestamp}_{duration}s_{motion}.{extension}
	filename := fmt.Sprintf("%s_%ss_%s.%s", timestamp, durationStr, motionStr, extension)
	filePath := filepath.Join(m.outputDir, filename)

	// Save the clip to disk
	if err := os.WriteFile(filePath, videoData, 0644); err != nil {
		log.Printf("[MOCK] Failed to save clip to %s: %v", filePath, err)
		return fmt.Errorf("failed to save clip: %w", err)
	}

	// Simulate upload time based on file size
	uploadTime := time.Duration(len(videoData)/1024) * time.Millisecond // 1ms per KB
	if uploadTime > 5*time.Second {
		uploadTime = 5 * time.Second // Cap at 5 seconds
	}
	time.Sleep(uploadTime)

	// Record the upload
	upload := UploadRecord{
		Timestamp: time.Now(),
		Size:      len(videoData),
		MimeType:  mimeType,
		FilePath:  filePath,
	}
	m.uploads = append(m.uploads, upload)

	log.Printf("[MOCK] Upload completed in %v. Saved to: %s (motion: %v). Total uploads: %d",
		uploadTime, filename, hasMotion, len(m.uploads))
	return nil
}

// GetUploads returns all recorded uploads (for testing purposes)
func (m *MockCaptureServerClient) GetUploads() []UploadRecord {
	return m.uploads
}

// GetOutputDirectory returns the directory where clips are saved
func (m *MockCaptureServerClient) GetOutputDirectory() string {
	return m.outputDir
}

// CleanupSavedClips removes all saved clip files (useful for testing cleanup)
func (m *MockCaptureServerClient) CleanupSavedClips() error {
	log.Printf("[MOCK] Cleaning up saved clips in %s", m.outputDir)

	for _, upload := range m.uploads {
		if err := os.Remove(upload.FilePath); err != nil && !os.IsNotExist(err) {
			log.Printf("[MOCK] Warning: Failed to remove %s: %v", upload.FilePath, err)
		}
	}

	// Clear the uploads list
	m.uploads = make([]UploadRecord, 0)
	log.Printf("[MOCK] Cleanup completed")
	return nil
}

// UpdateSettings allows changing settings during runtime (for testing dynamic updates)
func (m *MockCaptureServerClient) UpdateSettings(settings *models.ClientSettings) {
	log.Printf("[MOCK] Settings updated: MotionOnly=%v, Duration=%ds",
		settings.MotionOnly, settings.ClipDurationSeconds)
	m.settings = settings
}

// SimulateSettingsChange simulates a server-side settings change for testing
func (m *MockCaptureServerClient) SimulateSettingsChange() {
	log.Println("[MOCK] Simulating settings change...")

	// Toggle motion only mode and change duration
	m.settings.MotionOnly = !m.settings.MotionOnly

	// Cycle through different durations
	switch m.settings.ClipDurationSeconds {
	case 30:
		m.settings.ClipDurationSeconds = 60
	case 60:
		m.settings.ClipDurationSeconds = 15
	default:
		m.settings.ClipDurationSeconds = 30
	}

	// Cycle through resolutions
	switch m.settings.DownscaleResolution {
	case "720p":
		m.settings.DownscaleResolution = "480p"
	case "480p":
		m.settings.DownscaleResolution = "1080p"
	default:
		m.settings.DownscaleResolution = "720p"
	}

	log.Printf("[MOCK] New settings: MotionOnly=%v, Duration=%ds, Resolution=%s",
		m.settings.MotionOnly,
		m.settings.ClipDurationSeconds,
		m.settings.DownscaleResolution)
}
