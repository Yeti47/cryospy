package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/client"
	common "github.com/yeti47/cryospy/client/capture-client/common"
	"github.com/yeti47/cryospy/client/capture-client/config"
	filemanagement "github.com/yeti47/cryospy/client/capture-client/file-management"
	motiondetection "github.com/yeti47/cryospy/client/capture-client/motion-detection"
	postprocessing "github.com/yeti47/cryospy/client/capture-client/post-processing"
	"github.com/yeti47/cryospy/client/capture-client/recording"
	"github.com/yeti47/cryospy/client/capture-client/uploading"
)

func main() {
	// Parse command line flags
	clientID := flag.String("client-id", "", "Client ID (overrides config)")
	clientSecret := flag.String("client-secret", "", "Client secret (overrides config)")
	serverURL := flag.String("server-url", "", "Server URL (overrides config)")
	cameraDevice := flag.String("camera-device", "", "Camera device path (overrides config)")
	bufferSize := flag.Int("buffer-size", 0, "Buffer size (overrides config)")
	settingsSyncSeconds := flag.Int("settings-sync-seconds", 0, "Settings sync interval in seconds (overrides config)")
	serverTimeoutSeconds := flag.Int("server-timeout-seconds", 0, "Server timeout in seconds (overrides config)")

	flag.Parse()

	// Set up logging to both console and daily rotating file
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Create daily rotating writer
	rotatingWriter := common.NewDailyRotatingWriter(logDir, "capture-client")
	defer rotatingWriter.Close()

	// Set up multi-writer to write to both console and rotating file
	multiWriter := io.MultiWriter(os.Stdout, rotatingWriter)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Logging initialized - writing to console and daily rotating log files")

	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Apply CLI overrides if provided
	cfg.Override(config.ConfigOverrides{
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		ServerURL:            serverURL,
		CameraDevice:         cameraDevice,
		BufferSize:           bufferSize,
		SettingsSyncSeconds:  settingsSyncSeconds,
		ServerTimeoutSeconds: serverTimeoutSeconds,
	})

	// Log final configuration (without sensitive data)
	log.Printf("Configuration: ServerURL=%s, CameraDevice=%s, BufferSize=%d, SettingsSyncSeconds=%d",
		cfg.ServerURL, cfg.CameraDevice, cfg.BufferSize, cfg.SettingsSyncSeconds)

	// Set up temporary directory
	tempDir := filepath.Join(".", "temp")

	// Create dependencies
	log.Println("Setting up dependencies...")

	// Create server client
	// Create server client with bundled auth parameters
	clientAuth := client.ClientAuth{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
	}
	proxyAuth := client.ProxyAuth{
		Header: cfg.ProxyAuthHeader,
		Value:  cfg.ProxyAuthValue,
	}
	serverClient := client.NewCaptureServerClient(cfg.ServerURL, clientAuth, proxyAuth, time.Duration(cfg.ServerTimeoutSeconds)*time.Second)

	// Create client settings provider
	settingsCacheDuration := time.Duration(cfg.SettingsSyncSeconds) * time.Second
	clientSettingsProvider, err := config.NewClientSettingsProvider(serverClient, settingsCacheDuration)
	if err != nil {
		log.Fatalf("Failed to create client settings provider: %v", err)
	}

	// Create domain-specific settings providers
	motionSettingsProvider := motiondetection.NewMotionDetectionSettingsProvider(clientSettingsProvider)
	postProcessingSettingsProvider := postprocessing.NewPostProcessingSettingsProvider(clientSettingsProvider)
	recordingSettingsProvider := recording.NewRecordingSettingsProvider(clientSettingsProvider)

	// Create codec provider for video post-processing
	codecProvider := common.NewFFmpegCodecProvider()

	// Create core components
	recorder := recording.NewGoCVRecorder(cfg.CameraDevice, tempDir, recordingSettingsProvider)
	motionDetector := motiondetection.NewGoCVMotionDetector(motionSettingsProvider)
	// Create post-processor with codec provider
	postProcessor := postprocessing.NewFfmpegPostProcessor(postProcessingSettingsProvider, codecProvider)
	fileTracker := filemanagement.NewLocalFileTracker(tempDir)
	uploadQueue := uploading.NewUploadQueue(serverClient, cfg.BufferSize, time.Duration(cfg.ServerTimeoutSeconds)*time.Second+5*time.Second)

	// Create capture client
	captureClient := NewCaptureClient(
		recorder,
		motionDetector,
		postProcessor,
		uploadQueue,
		fileTracker,
		clientSettingsProvider,
		cfg.CameraDevice,
	)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the capture client
	log.Println("Starting capture client...")
	if err := captureClient.Start(); err != nil {
		log.Fatalf("Failed to start capture client: %v", err)
	}

	// Wait for shutdown signal
	log.Println("Capture client running. Press Ctrl+C to stop.")
	<-sigChan
	log.Println("Shutdown signal received, stopping capture...")

	// Stop the capture client
	if err := captureClient.Stop(); err != nil {
		log.Printf("Error stopping capture client: %v", err)
	}

	log.Println("Capture client stopped")
}
