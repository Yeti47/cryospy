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
	motionSensitivity := flag.Float64("motion-sensitivity", 0, "Motion detection sensitivity as percentage (overrides config, e.g., 1.0, 0.5)")

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
		MotionSensitivity:   motionSensitivity,
	})

	// Log final configuration (without sensitive data)
	log.Printf("Configuration: ServerURL=%s, CameraDevice=%s, BufferSize=%d, SettingsSyncSeconds=%d",
		cfg.ServerURL, cfg.CameraDevice, cfg.BufferSize, cfg.SettingsSyncSeconds)
	log.Printf("Video Processing: Codec=%s, Format=%s, BitRate=%s, CaptureCodec=%s, CaptureFrameRate=%.1f",
		cfg.VideoCodec, cfg.VideoOutputFormat, cfg.VideoBitRate, cfg.CaptureCodec, cfg.CaptureFrameRate)

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

	// Create capture application
	app := &CaptureApp{
		config:         cfg,
		settings:       settings,
		clientService:  clientService,
		processor:      processor,
		uploadQueue:    make(chan string, cfg.BufferSize), // Use configured buffer size
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
	uploadQueue    chan string
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
func (app *CaptureApp) onChunkReady(chunkPath string) {
	log.Printf("Chunk ready: %s", chunkPath)

	// Process the chunk in a separate goroutine to not block capture
	go app.processChunk(chunkPath)
}

// processChunk handles the processing and upload of a video chunk
func (app *CaptureApp) processChunk(rawChunkPath string) {
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

	// Process the chunk (compress, downscale, etc.)
	// Create proper output filename by replacing extension
	baseName := strings.TrimSuffix(rawChunkPath, ".mp4")
	processedPath := baseName + "_processed.mp4"
	err = app.processor.ProcessClip(rawChunkPath, processedPath, settings)
	if err != nil {
		log.Printf("Failed to process chunk %s: %v", rawChunkPath, err)
		return
	}

	// Queue for upload
	select {
	case app.uploadQueue <- processedPath:
		log.Printf("Queued %s for upload", processedPath)
	default:
		log.Printf("Upload queue full, dropping %s", processedPath)
		os.Remove(processedPath)
	}
}

// uploadWorker handles uploading processed video chunks
func (app *CaptureApp) uploadWorker() {
	for {
		select {
		case processedPath := <-app.uploadQueue:
			app.uploadChunk(processedPath)
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

	// Check if settings have changed
	app.settingsMu.Lock()
	defer app.settingsMu.Unlock()

	if app.settingsChanged(app.settings, newSettings) {
		log.Println("Client settings have changed, updating...")
		oldSettings := app.settings
		app.settings = newSettings

		log.Printf("Settings updated - Motion only mode: %v, Chunk duration: %v",
			app.settings.MotionOnly, app.settings.ChunkDuration())

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

	return old.ID != new.ID ||
		old.StorageLimitMegabytes != new.StorageLimitMegabytes ||
		old.ClipDurationSeconds != new.ClipDurationSeconds ||
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
func (app *CaptureApp) uploadChunk(processedPath string) {
	defer os.Remove(processedPath) // Clean up after upload

	log.Printf("Uploading %s...", processedPath)

	// Read the video file
	videoData, err := os.ReadFile(processedPath)
	if err != nil {
		log.Printf("Failed to read processed video %s: %v", processedPath, err)
		return
	}

	// Upload to server
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = app.clientService.UploadClip(ctx, videoData, "video/mp4")
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

	// Wait for any processing to complete
	for {
		app.mu.Lock()
		processing := app.processingClip
		queueEmpty := len(app.uploadQueue) == 0
		app.mu.Unlock()

		if !processing && queueEmpty {
			break
		}

		log.Println("Waiting for processing and uploads to complete...")
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Capture application stopped")
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
