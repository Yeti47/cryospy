package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/config"
	"github.com/yeti47/cryospy/client/capture-client/models"
	"github.com/yeti47/cryospy/client/capture-client/video"
)

// UploadInfo contains information needed for uploading a video clip
type UploadInfo struct {
	FilePath           string
	HasMotion          bool
	MimeType           string
	Duration           time.Duration
	RecordingTimestamp time.Time
}

func main() {
	// Parse command line flags
	testMode := flag.Bool("test", false, "Run in test mode with mock server client")

	// Config override flags
	clientID := flag.String("client-id", "", "Client ID (overrides config)")
	clientSecret := flag.String("client-secret", "", "Client secret (overrides config)")
	serverURL := flag.String("server-url", "", "Server URL (overrides config)")
	cameraDevice := flag.String("camera-device", "", "Camera device path (overrides config)")
	bufferSize := flag.Int("buffer-size", 0, "Buffer size (overrides config)")
	settingsSyncSeconds := flag.Int("settings-sync-seconds", 0, "Settings sync interval in seconds (overrides config)")

	// Video processing override flags
	videoCodec := flag.String("video-codec", "", "Video codec for processing (overrides config, e.g., 'mpeg4', 'libopenh264')")
	videoOutputFormat := flag.String("video-output-format", "", "Video output format (overrides config, e.g., 'mp4', 'avi')")
	videoBitRate := flag.String("video-bitrate", "", "Video bitrate (overrides config, e.g., '500k', '1M')")
	captureCodec := flag.String("capture-codec", "", "Capture codec (overrides config, e.g., 'MJPG', 'MP4V')")
	captureFrameRate := flag.Float64("capture-framerate", 0, "Capture frame rate (overrides config, e.g., 15.0, 30.0)")
	motionMinArea := flag.Int("motion-min-area", 0, "Minimum contour area for motion detection (overrides config)")

	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Apply CLI overrides if provided
	cfg.Override(config.ConfigOverrides{
		ClientID:            clientID,
		ClientSecret:        clientSecret,
		ServerURL:           serverURL,
		CameraDevice:        cameraDevice,
		BufferSize:          bufferSize,
		SettingsSyncSeconds: settingsSyncSeconds,
		VideoCodec:          videoCodec,
		VideoOutputFormat:   videoOutputFormat,
		VideoBitRate:        videoBitRate,
		CaptureCodec:        captureCodec,
		CaptureFrameRate:    captureFrameRate,
		MotionMinArea:       motionMinArea,
	})

	// Log final configuration (without sensitive data)
	log.Printf("Configuration: ServerURL=%s, CameraDevice=%s, BufferSize=%d, SettingsSyncSeconds=%d",
		cfg.ServerURL, cfg.CameraDevice, cfg.BufferSize, cfg.SettingsSyncSeconds)
	log.Printf("Video Processing: Codec=%s, Format=%s, BitRate=%s, CaptureCodec=%s, CaptureFrameRate=%.1f, MotionMinArea=%d",
		cfg.VideoCodec, cfg.VideoOutputFormat, cfg.VideoBitRate, cfg.CaptureCodec, cfg.CaptureFrameRate, cfg.MotionMinArea)

	// Create client service based on mode
	var clientService client.CaptureServerClient
	var mockClient *client.MockCaptureServerClient
	if *testMode {
		log.Println("Running in TEST MODE with mock server client")
		mockClient = client.NewMockCaptureServerClient()
		clientService = mockClient

		// In test mode, also start a goroutine to simulate settings changes
		go simulateSettingsChanges(mockClient)
	} else {
		log.Println("Running in PRODUCTION MODE with real server client")
		clientService = client.NewCaptureServerClient(cfg.ServerURL, cfg.ClientID, cfg.ClientSecret, 30*time.Second)
	}

	// Fetch client settings from server
	log.Println("Fetching client settings from server...")
	settings, err := clientService.GetClientSettings(context.Background())
	if err != nil {
		log.Fatalf("Failed to fetch client settings: %v", err)
	}
	log.Printf("Client settings loaded: Motion only mode: %v, Chunk duration: %v",
		settings.MotionOnly, settings.ChunkDuration())

	// Create video processor
	tempDir := filepath.Join(".", "temp")
	processor, err := video.NewVideoProcessor(tempDir, cfg)
	if err != nil {
		log.Fatalf("Failed to create video processor: %v", err)
	}

	// Clean up old temporary files on startup
	log.Println("Cleaning up old temporary files...")
	if err := processor.CleanupTempFiles(); err != nil {
		log.Printf("Warning: Failed to cleanup temp files: %v", err)
	}

	// Create capture application
	app := &CaptureApp{
		config:         cfg,
		settings:       settings,
		clientService:  clientService,
		processor:      processor,
		uploadQueue:    make(chan UploadInfo, cfg.BufferSize), // Use configured buffer size
		processingClip: false,
		shutdown:       make(chan struct{}),
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the application
	log.Println("Starting capture client...")

	// Start upload worker
	go app.uploadWorker()

	// Start settings sync worker
	go app.settingsSyncWorker()

	// Start continuous capture
	err = app.startCapture()
	if err != nil {
		log.Fatalf("Failed to start capture: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping capture...")
	app.stop()

	log.Println("Capture client stopped")
}

// CaptureApp manages the video capture application
type CaptureApp struct {
	config         *config.Config
	settings       *models.ClientSettings
	settingsMu     sync.RWMutex // Protects settings access
	clientService  client.CaptureServerClient
	processor      video.Processor
	uploadQueue    chan UploadInfo
	processingClip bool
	mu             sync.Mutex
	shutdown       chan struct{}
}

// startCapture begins continuous video capture
func (app *CaptureApp) startCapture() error {
	settings := app.getSettings()
	log.Printf("Starting continuous capture from device %s with %v chunks",
		app.config.CameraDevice, settings.ChunkDuration())

	// Start continuous capture
	return app.processor.StartContinuousCapture(
		app.config.CameraDevice,
		settings.ChunkDuration(),
		app.onChunkReady,
	)
}

// onChunkReady is called when a video chunk is ready for processing
func (app *CaptureApp) onChunkReady(videoClip *models.VideoClip) {
	log.Printf("Chunk ready: %s (recorded at %v)", videoClip.FilePath, videoClip.Timestamp)

	// Process the chunk in a separate goroutine to not block capture
	go app.processChunk(videoClip)
}

// processChunk handles the processing and upload of a video chunk
func (app *CaptureApp) processChunk(videoClip *models.VideoClip) {
	rawChunkPath := videoClip.FilePath
	recordingTimestamp := videoClip.Timestamp

	// Check if we're shutting down before starting processing
	select {
	case <-app.shutdown:
		log.Printf("Skipping processing of %s due to shutdown", rawChunkPath)
		os.Remove(rawChunkPath)
		return
	default:
	}

	app.mu.Lock()
	app.processingClip = true
	app.mu.Unlock()

	defer func() {
		app.mu.Lock()
		app.processingClip = false
		app.mu.Unlock()

		// Clean up raw chunk file
		os.Remove(rawChunkPath)
	}()

	// Get current settings (thread-safe)
	settings := app.getSettings()

	// Always perform motion detection to determine if we should upload
	hasMotion, err := app.processor.DetectMotion(rawChunkPath)
	if err != nil {
		log.Printf("Motion detection failed for %s: %v", rawChunkPath, err)
		// If motion detection fails, proceed with upload to be safe
		hasMotion = true
	}

	// If motion-only mode is enabled and no motion detected, skip upload
	if settings.MotionOnly && !hasMotion {
		log.Printf("Motion-only mode: No motion detected in %s, skipping upload", rawChunkPath)
		return
	}

	if hasMotion {
		log.Printf("Motion detected in %s", rawChunkPath)
	} else {
		log.Printf("No motion detected in %s, but uploading (motion-only mode disabled)", rawChunkPath)
	}

	// Check if the recorded file is valid before processing
	if stat, err := os.Stat(rawChunkPath); err != nil {
		log.Printf("Failed to stat chunk file %s: %v", rawChunkPath, err)
		return
	} else if stat.Size() < 1000 { // Minimum reasonable size for a video file
		log.Printf("Chunk file %s is too small (%d bytes), skipping", rawChunkPath, stat.Size())
		return
	}

	// Check again if we're shutting down before expensive processing
	select {
	case <-app.shutdown:
		log.Printf("Skipping processing of %s due to shutdown", rawChunkPath)
		return
	default:
	}

	// Process the chunk (compress, downscale, etc.)
	// Create proper output filename using the configured output format
	captureExtension := app.processor.GetCaptureFileExtension()
	baseName := strings.TrimSuffix(rawChunkPath, captureExtension)
	outputExtension := app.processor.GetOutputFileExtension()
	processedPath := baseName + "_processed" + outputExtension
	err = app.processor.ProcessClip(rawChunkPath, processedPath, settings)
	if err != nil {
		log.Printf("Failed to process chunk %s: %v", rawChunkPath, err)
		return
	}

	// Get the actual duration from the processed video metadata
	actualDuration, err := app.processor.GetVideoDuration(processedPath)
	if err != nil {
		log.Printf("Failed to get video duration for %s: %v, using settings duration as fallback", processedPath, err)
		actualDuration = settings.ChunkDuration()
	}

	// Queue for upload
	uploadInfo := UploadInfo{
		FilePath:           processedPath,
		HasMotion:          hasMotion,
		MimeType:           app.processor.GetOutputMimeType(),
		Duration:           actualDuration,
		RecordingTimestamp: recordingTimestamp,
	}
	select {
	case app.uploadQueue <- uploadInfo:
		log.Printf("Queued %s for upload (motion: %v, duration: %v, type: %s)", processedPath, hasMotion, actualDuration, uploadInfo.MimeType)
	default:
		log.Printf("Upload queue full, dropping %s", processedPath)
		os.Remove(processedPath)
	}
}

// uploadWorker handles uploading processed video chunks
func (app *CaptureApp) uploadWorker() {
	for {
		select {
		case uploadInfo := <-app.uploadQueue:
			app.uploadChunk(uploadInfo.FilePath, uploadInfo.HasMotion, uploadInfo.MimeType, uploadInfo.Duration, uploadInfo.RecordingTimestamp)
		case <-app.shutdown:
			return
		}
	}
}

// settingsSyncWorker periodically syncs client settings from the server
func (app *CaptureApp) settingsSyncWorker() {
	// Use configurable sync interval (default 300 seconds = 5 minutes)
	syncInterval := time.Duration(app.config.SettingsSyncSeconds) * time.Second
	log.Printf("Starting settings sync worker with %v interval", syncInterval)

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.syncSettings()
		case <-app.shutdown:
			return
		}
	}
}

// syncSettings fetches and updates client settings from the server
func (app *CaptureApp) syncSettings() {
	log.Println("Syncing client settings from server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newSettings, err := app.clientService.GetClientSettings(ctx)
	if err != nil {
		log.Printf("Failed to sync client settings: %v", err)
		return
	}

	// Get current settings for comparison (minimize lock time)
	app.settingsMu.RLock()
	currentSettings := app.settings
	app.settingsMu.RUnlock()

	// Check if settings have changed (outside of lock)
	if app.settingsChanged(currentSettings, newSettings) {
		log.Println("Client settings have changed, updating...")

		// Only hold the write lock for the actual update
		app.settingsMu.Lock()
		oldSettings := app.settings
		app.settings = newSettings
		app.settingsMu.Unlock()

		log.Printf("Settings updated - Motion only mode: %v, Chunk duration: %v",
			newSettings.MotionOnly, newSettings.ChunkDuration())

		// If chunk duration changed, we might need to restart capture
		// For now, just log it - a full restart would be complex
		if oldSettings.ChunkDuration() != newSettings.ChunkDuration() {
			log.Printf("Warning: Chunk duration changed from %v to %v. This will take effect on the next capture restart.",
				oldSettings.ChunkDuration(), newSettings.ChunkDuration())
		}
	}
}

// settingsChanged compares two ClientSettings and returns true if they differ
func (app *CaptureApp) settingsChanged(old, new *models.ClientSettings) bool {
	if old == nil || new == nil {
		return true
	}
	// Compare all relevant fields (we don't care about the ID and the storage capacity here)
	return old.ClipDurationSeconds != new.ClipDurationSeconds ||
		old.MotionOnly != new.MotionOnly ||
		old.Grayscale != new.Grayscale ||
		old.DownscaleResolution != new.DownscaleResolution
}

// getSettings safely returns a copy of the current settings
func (app *CaptureApp) getSettings() *models.ClientSettings {
	app.settingsMu.RLock()
	defer app.settingsMu.RUnlock()

	// Return a copy to avoid race conditions
	settingsCopy := *app.settings
	return &settingsCopy
}

// uploadChunk uploads a processed video chunk to the server
func (app *CaptureApp) uploadChunk(processedPath string, hasMotion bool, mimeType string, duration time.Duration, recordingTimestamp time.Time) {
	defer os.Remove(processedPath) // Clean up after upload

	log.Printf("Uploading %s (motion: %v, duration: %v, type: %s, recorded: %v)...", processedPath, hasMotion, duration, mimeType, recordingTimestamp)

	// Read the video file
	videoData, err := os.ReadFile(processedPath)
	if err != nil {
		log.Printf("Failed to read processed video %s: %v", processedPath, err)
		return
	}

	// Upload to server
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use the actual recording timestamp instead of upload time
	err = app.clientService.UploadClip(ctx, videoData, mimeType, duration, hasMotion, recordingTimestamp)
	if err != nil {
		log.Printf("Failed to upload %s: %v", processedPath, err)
		// TODO: Implement retry logic or save to disk for later retry
		return
	}

	log.Printf("Successfully uploaded %s", processedPath)
}

// stop gracefully stops the capture application
func (app *CaptureApp) stop() {
	log.Println("Stopping video processor...")
	app.processor.StopContinuousCapture()

	log.Println("Stopping upload worker...")
	close(app.shutdown)

	// Wait for any processing to complete with a timeout to avoid deadlocks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Poll for completion with timeout
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	//nolint:S1000 // This pattern is intentional for graceful shutdown
	for {
		select {
		case <-ctx.Done():
			log.Println("Timeout waiting for processing to complete, forcing shutdown")
			return
		case <-ticker.C:
			// Check if processing is complete
			app.mu.Lock()
			processing := app.processingClip
			queueEmpty := len(app.uploadQueue) == 0
			app.mu.Unlock()

			if !processing && queueEmpty {
				log.Println("Capture application stopped")
				return
			}

			log.Println("Waiting for processing and uploads to complete...")
		}
	}
}

// simulateSettingsChanges simulates server-side settings changes for testing
func simulateSettingsChanges(mockClient *client.MockCaptureServerClient) {
	// Wait a bit before starting to simulate changes
	time.Sleep(2 * time.Minute)

	// Simulate settings changes every 3 minutes
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mockClient.SimulateSettingsChange()
		}
	}
}
