package video

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xfrr/goffmpeg/transcoder"
	"github.com/yeti47/cryospy/client/capture-client/config"
	"github.com/yeti47/cryospy/client/capture-client/models"
	"gocv.io/x/gocv"
)

// Processor handles video recording and processing operations
type Processor interface {
	RecordClip(device string, duration time.Duration, outputPath string) error
	ProcessClip(inputPath, outputPath string, settings *models.ClientSettings) error
	DetectMotion(videoPath string) (bool, error)
	StartContinuousCapture(device string, chunkDuration time.Duration, onChunkReady func(string)) error
	StopContinuousCapture()
}

// VideoProcessor implements Processor using gocv for capture/motion detection and goffmpeg for processing
type VideoProcessor struct {
	tempDir     string
	capturing   bool
	stopCapture chan bool
	webcam      *gocv.VideoCapture
	config      *config.Config // Add config reference
}

// NewVideoProcessor creates a new hybrid processor
func NewVideoProcessor(tempDir string, cfg *config.Config) (Processor, error) {
	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &VideoProcessor{
		tempDir:     tempDir,
		capturing:   false,
		stopCapture: make(chan bool, 1),
		config:      cfg,
	}, nil
}

// RecordClip records a video clip from the specified device using gocv
func (p *VideoProcessor) RecordClip(device string, duration time.Duration, outputPath string) error {
	// Parse device ID (assume it's an integer for webcam device)
	deviceID := 0
	if device != "" && device != "0" {
		if id, err := strconv.Atoi(device); err == nil {
			deviceID = id
		}
	}

	// Open webcam
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		return fmt.Errorf("failed to open webcam: %w", err)
	}
	defer webcam.Close()

	// Create video writer - use MJPG codec which goffmpeg can convert later
	// Set isColor to false to disable audio (gocv doesn't capture audio anyway)
	writer, err := gocv.VideoWriterFile(outputPath, "MJPG", 30.0, 640, 480, false)
	if err != nil {
		return fmt.Errorf("failed to create video writer: %w", err)
	}
	defer writer.Close()

	// Record for the specified duration
	img := gocv.NewMat()
	defer img.Close()

	startTime := time.Now()
	for time.Since(startTime) < duration {
		if ok := webcam.Read(&img); !ok {
			return fmt.Errorf("failed to read from webcam")
		}

		if img.Empty() {
			continue
		}

		writer.Write(img)

		// Small delay to control frame rate
		time.Sleep(time.Millisecond * 33) // ~30 FPS
	}

	return nil
}

// ProcessClip processes a video clip according to client settings using goffmpeg
func (p *VideoProcessor) ProcessClip(inputPath, outputPath string, settings *models.ClientSettings) error {
	// Create transcoder instance
	trans := new(transcoder.Transcoder)

	// Initialize transcoder with input and output files
	err := trans.Initialize(inputPath, outputPath)
	if err != nil {
		return fmt.Errorf("failed to initialize transcoder: %w", err)
	}

	// Configure basic output settings - video only, no audio
	trans.MediaFile().SetVideoCodec(p.config.VideoCodec)
	trans.MediaFile().SetOutputFormat(p.config.VideoOutputFormat)

	// Disable audio streams
	trans.MediaFile().SetSkipAudio(true)

	// Build video filters
	var filters []string

	// Apply grayscale filter if enabled
	if settings.Grayscale {
		filters = append(filters, "format=gray")
	}

	// Apply downscaling if specified
	if settings.DownscaleResolution != "" {
		resolution := parseResolution(settings.DownscaleResolution)
		if resolution != "" {
			filters = append(filters, fmt.Sprintf("scale=%s", resolution))
		}
	}

	// Apply filters if any
	if len(filters) > 0 {
		filterChain := strings.Join(filters, ",")
		trans.MediaFile().SetVideoFilter(filterChain)
	}

	// Apply compression settings for smaller file size (video only)
	trans.MediaFile().SetVideoBitRate(p.config.VideoBitRate) // Use configured bitrate

	// Run the transcoding
	done := trans.Run(true) // true = progress reporting
	err = <-done

	if err != nil {
		return fmt.Errorf("failed to process video: %w", err)
	}

	return nil
}

// DetectMotion analyzes a video file for motion using gocv
func (p *VideoProcessor) DetectMotion(videoPath string) (bool, error) {
	// Open video file
	video, err := gocv.OpenVideoCapture(videoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open video file: %w", err)
	}
	defer video.Close()

	// Create motion detector
	detector := gocv.NewBackgroundSubtractorMOG2()
	defer detector.Close()

	img := gocv.NewMat()
	defer img.Close()

	fgMask := gocv.NewMat()
	defer fgMask.Close()

	motionDetected := false
	frameCount := 0
	maxFramesToCheck := 100 // Limit frames to check for performance

	for frameCount < maxFramesToCheck {
		if ok := video.Read(&img); !ok {
			break // End of video
		}

		if img.Empty() {
			continue
		}

		// Apply background subtraction
		detector.Apply(img, &fgMask)

		// Count non-zero pixels (motion pixels)
		nonZeroPixels := gocv.CountNonZero(fgMask)

		// Calculate motion threshold based on configured sensitivity
		// MotionSensitivity is a percentage (e.g., 1.0 = 1% of pixels)
		motionThreshold := int(float64(img.Rows()*img.Cols()) * p.config.MotionSensitivity / 100.0)
		if nonZeroPixels > motionThreshold {
			motionDetected = true
			break
		}

		frameCount++
	}

	return motionDetected, nil
}

// StartContinuousCapture starts continuous video capture with chunking
func (p *VideoProcessor) StartContinuousCapture(device string, chunkDuration time.Duration, onChunkReady func(string)) error {
	if p.capturing {
		return fmt.Errorf("capture already in progress")
	}

	// Parse device ID
	deviceID := 0
	if device != "" && device != "0" {
		if id, err := strconv.Atoi(device); err == nil {
			deviceID = id
		}
	}

	// Open webcam
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		return fmt.Errorf("failed to open webcam: %w", err)
	}

	p.webcam = webcam
	p.capturing = true

	// Start capture routine
	go p.captureLoop(chunkDuration, onChunkReady)

	return nil
}

// StopContinuousCapture stops the continuous capture
func (p *VideoProcessor) StopContinuousCapture() {
	if !p.capturing {
		return
	}

	fmt.Println("Stopping continuous capture...")
	p.capturing = false

	// Send stop signal with timeout
	select {
	case p.stopCapture <- true:
		fmt.Println("Stop signal sent successfully")
	case <-time.After(1 * time.Second):
		fmt.Println("Timeout sending stop signal, forcing cleanup")
	}

	// Force close webcam if open
	if p.webcam != nil {
		fmt.Println("Closing webcam...")
		p.webcam.Close()
		p.webcam = nil
		fmt.Println("Webcam closed")
	}
}

// captureLoop handles the continuous capture process
func (p *VideoProcessor) captureLoop(chunkDuration time.Duration, onChunkReady func(string)) {
	defer func() {
		fmt.Println("Capture loop ending, cleaning up...")
		p.capturing = false
		if p.webcam != nil {
			fmt.Println("Closing webcam in capture loop...")
			p.webcam.Close()
			p.webcam = nil
			fmt.Println("Webcam closed in capture loop")
		}
	}()

	chunkCount := 0
	for p.capturing {
		// Create unique filename for this chunk
		chunkCount++
		chunkFile := fmt.Sprintf("%s/chunk_%d_%d.mp4", p.tempDir, time.Now().Unix(), chunkCount)

		// Record chunk
		err := p.recordChunk(chunkFile, chunkDuration)
		if err != nil {
			fmt.Printf("Error recording chunk: %v\n", err)
			// Don't continue immediately, check for stop signal first
		} else {
			// Notify that chunk is ready only if recording was successful
			if onChunkReady != nil {
				onChunkReady(chunkFile)
			}
		}

		// Check if we should stop (always check after each iteration)
		select {
		case <-p.stopCapture:
			fmt.Println("Stop signal received in capture loop")
			return
		case <-time.After(100 * time.Millisecond): // Short pause to prevent busy loop
		}
	}
}

// recordChunk records a single chunk using the opened webcam
func (p *VideoProcessor) recordChunk(outputPath string, duration time.Duration) error {
	if p.webcam == nil {
		return fmt.Errorf("webcam not initialized")
	}

	// Get frame properties from webcam
	width := int(p.webcam.Get(gocv.VideoCaptureFrameWidth))
	height := int(p.webcam.Get(gocv.VideoCaptureFrameHeight))

	if width <= 0 || height <= 0 {
		width = 640
		height = 480
		fmt.Printf("Using default resolution: %dx%d\n", width, height)
	} else {
		fmt.Printf("Using webcam resolution: %dx%d\n", width, height)
	}

	// Create video writer - use configured codec
	writer, err := gocv.VideoWriterFile(outputPath, p.config.CaptureCodec, p.config.CaptureFrameRate, width, height, true)
	if err != nil {
		return fmt.Errorf("failed to create video writer: %w", err)
	}
	defer func() {
		fmt.Printf("Closing video writer for %s\n", outputPath)
		writer.Close()
	}()

	img := gocv.NewMat()
	defer img.Close()

	startTime := time.Now()
	frameCount := 0

recordLoop:
	for time.Since(startTime) < duration && p.capturing {
		if ok := p.webcam.Read(&img); !ok {
			fmt.Printf("Failed to read frame %d from webcam\n", frameCount)
			time.Sleep(time.Millisecond * 67) // Wait a bit before retrying
			continue
		}

		if img.Empty() {
			fmt.Printf("Empty frame %d from webcam\n", frameCount)
			continue
		}

		// Write frame to video
		if err := writer.Write(img); err != nil {
			fmt.Printf("Failed to write frame %d to video: %v\n", frameCount, err)
		}
		frameCount++

		// Control frame rate (~15 FPS for smaller files)
		time.Sleep(time.Millisecond * 67)

		// Check for stop signal
		select {
		case <-p.stopCapture:
			break recordLoop
		default:
		}
	}

	fmt.Printf("Recorded %d frames for %s in %v\n", frameCount, outputPath, time.Since(startTime))

	// Check if we recorded any frames
	if frameCount == 0 {
		return fmt.Errorf("no frames were recorded from webcam")
	}

	return nil
}

// parseResolution converts resolution strings like "720p" to ffmpeg scale format
func parseResolution(resolution string) string {
	resolution = strings.ToLower(strings.TrimSpace(resolution))

	switch resolution {
	case "360p":
		return "640:360"
	case "480p":
		return "854:480"
	case "720p":
		return "1280:720"
	case "1080p":
		return "1920:1080"
	default:
		// Try to parse custom resolution like "1280x720"
		if strings.Contains(resolution, "x") {
			parts := strings.Split(resolution, "x")
			if len(parts) == 2 {
				if width, err := strconv.Atoi(parts[0]); err == nil {
					if height, err := strconv.Atoi(parts[1]); err == nil {
						return fmt.Sprintf("%d:%d", width, height)
					}
				}
			}
		}
		return ""
	}
}
