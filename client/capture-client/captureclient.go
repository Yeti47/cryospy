package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/config"
	filemanagement "github.com/yeti47/cryospy/client/capture-client/file-management"
	motiondetection "github.com/yeti47/cryospy/client/capture-client/motion-detection"
	postprocessing "github.com/yeti47/cryospy/client/capture-client/post-processing"
	"github.com/yeti47/cryospy/client/capture-client/recording"
	"github.com/yeti47/cryospy/client/capture-client/uploading"
)

// CaptureClient orchestrates video recording, processing, and uploading
type CaptureClient struct {
	// Core components
	recorder               recording.Recorder
	motionDetector         motiondetection.MotionDetector
	postProcessor          postprocessing.PostProcessor
	uploadQueue            uploading.UploadQueue
	fileTracker            filemanagement.FileTracker
	clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]

	// Configuration
	cameraDevice string

	// State management
	isRunning    bool
	mu           sync.RWMutex
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// NewCaptureClient creates a new capture client with injected dependencies
func NewCaptureClient(
	recorder recording.Recorder,
	motionDetector motiondetection.MotionDetector,
	postProcessor postprocessing.PostProcessor,
	uploadQueue uploading.UploadQueue,
	fileTracker filemanagement.FileTracker,
	clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse],
	cameraDevice string,
) *CaptureClient {
	return &CaptureClient{
		recorder:               recorder,
		motionDetector:         motionDetector,
		postProcessor:          postProcessor,
		uploadQueue:            uploadQueue,
		fileTracker:            fileTracker,
		clientSettingsProvider: clientSettingsProvider,
		cameraDevice:           cameraDevice,
		shutdownChan:           make(chan struct{}),
	}
}

// Start begins the capture, processing, and upload workflow
func (c *CaptureClient) Start() error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return fmt.Errorf("capture client is already running")
	}
	c.isRunning = true
	c.mu.Unlock()

	log.Println("Starting capture client...")

	// Ensure temp directory exists
	if err := c.fileTracker.EnsureTempDirectory(); err != nil {
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Start upload service with success and failure callbacks to cleanup files
	c.wg.Add(1)
	go c.uploadQueue.Start(c.shutdownChan, &c.wg,
		func(job *uploading.UploadJob) {
			// Success callback - delete file after successful upload
			c.fileTracker.DeleteFile(job.FilePath)
		},
		func(job *uploading.UploadJob) {
			// Failure callback - delete file after all retries exhausted
			log.Printf("Upload permanently failed for %s, cleaning up file", job.FilePath)
			c.fileTracker.DeleteFile(job.FilePath)
		},
	)

	// Start recording with callback
	started, err := c.recorder.StartRecording(c.onRawClipReady, c.onRecordingError)
	if err != nil {
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
		close(c.shutdownChan)
		return fmt.Errorf("failed to start recording: %w", err)
	}

	if !started {
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
		close(c.shutdownChan)
		return fmt.Errorf("recording did not start (recorder busy)")
	}

	log.Println("Capture client started successfully")
	return nil
}

// Stop gracefully stops the capture client
func (c *CaptureClient) Stop() error {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return nil
	}
	c.isRunning = false
	c.mu.Unlock()

	log.Println("Stopping capture client...")

	// Stop recording
	if err := c.recorder.StopRecording(); err != nil {
		log.Printf("Error stopping recorder: %v", err)
	}

	// Signal shutdown to workers
	close(c.shutdownChan)

	// Wait for workers to finish
	c.wg.Wait()

	// Final cleanup of any remaining files in temp directory
	c.fileTracker.CleanupTempDirectory()

	log.Println("Capture client stopped")
	return nil
}

// IsRunning returns whether the capture client is currently running
func (c *CaptureClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// onRawClipReady is called when a raw video clip has been recorded
func (c *CaptureClient) onRawClipReady(rawClip *recording.RawClip) error {
	log.Printf("Raw clip ready: %s", rawClip.Path)

	// Process the clip asynchronously to not block recording
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.processRawClip(rawClip)
	}()

	return nil
}

// onRecordingError is called when a recording error occurs
func (c *CaptureClient) onRecordingError(err error) bool {
	log.Printf("Recording error: %v", err)

	// Check if we should continue or cancel
	c.mu.RLock()
	running := c.isRunning
	c.mu.RUnlock()

	if !running {
		return true // Cancel recording if we're shutting down
	}

	// Continue recording for transient errors
	return false
}

// processRawClip handles the complete processing workflow for a raw clip
func (c *CaptureClient) processRawClip(rawClip *recording.RawClip) {
	// Check if we're shutting down
	select {
	case <-c.shutdownChan:
		log.Printf("Skipping processing of %s due to shutdown", rawClip.Path)
		c.fileTracker.DeleteFile(rawClip.Path)
		return
	default:
	}

	// Get current client settings
	clientSettings := c.clientSettingsProvider.GetSettings()

	// Perform motion detection
	hasMotion, err := c.motionDetector.DetectMotion(rawClip.Path)
	if err != nil {
		log.Printf("Motion detection failed for %s: %v", rawClip.Path, err)
		// If motion detection fails, assume motion for safety
		hasMotion = true
	}

	// Check if we should skip upload based on motion-only setting
	if clientSettings.MotionOnly && !hasMotion {
		log.Printf("Motion-only mode: No motion detected in %s, skipping upload", rawClip.Path)
		c.fileTracker.DeleteFile(rawClip.Path)
		return
	}

	if hasMotion {
		log.Printf("Motion detected in %s", rawClip.Path)
	} else {
		log.Printf("No motion detected in %s, but uploading (motion-only mode disabled)", rawClip.Path)
	}

	// Check again if we're shutting down before expensive processing
	select {
	case <-c.shutdownChan:
		log.Printf("Skipping processing of %s due to shutdown", rawClip.Path)
		c.fileTracker.DeleteFile(rawClip.Path)
		return
	default:
	}

	// Post-process the video
	processedClip, err := c.postProcessor.ProcessVideo(rawClip)
	if err != nil {
		log.Printf("Post-processing failed for %s: %v", rawClip.Path, err)
		c.fileTracker.DeleteFile(rawClip.Path)
		return
	}

	// Create upload job
	uploadJob := &uploading.UploadJob{
		FilePath:           processedClip.Path,
		HasMotion:          hasMotion,
		Duration:           processedClip.Duration,
		RecordingTimestamp: rawClip.Timestamp,
		Format:             processedClip.Format,
	}

	// Queue for upload
	if c.uploadQueue.Queue(uploadJob) {
		log.Printf("Queued %s for upload (motion: %v, duration: %v)",
			processedClip.Path, hasMotion, processedClip.Duration)
		// Upload service will call cleanup callback after successful upload
	} else {
		// Upload queue full, clean up processed file
		log.Printf("Upload queue full, dropping %s", processedClip.Path)
		c.fileTracker.DeleteFile(processedClip.Path)
	}

	// Clean up raw clip
	c.fileTracker.DeleteFile(rawClip.Path)
}
