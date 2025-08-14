package video

import (
	"fmt"
	"image"
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
	StartContinuousCapture(device string, chunkDuration time.Duration, onChunkReady func(*models.VideoClip)) error
	StopContinuousCapture()
	CleanupTempFiles(maxAge time.Duration) error
	GetOutputMimeType() string
	GetOutputFileExtension() string
	GetCaptureFileExtension() string
	GetVideoDuration(videoPath string) (time.Duration, error)
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

// GetOutputMimeType returns the MIME type for the configured video output format
func (p *VideoProcessor) GetOutputMimeType() string {
	format := strings.ToLower(p.config.VideoOutputFormat)
	switch format {
	case "mp4":
		return "video/mp4"
	case "avi":
		return "video/x-msvideo"
	case "mkv":
		return "video/x-matroska"
	case "webm":
		return "video/webm"
	case "mov":
		return "video/quicktime"
	default:
		// Default to mp4 if unknown format
		return "video/mp4"
	}
}

// GetOutputFileExtension returns the file extension for the configured video output format
func (p *VideoProcessor) GetOutputFileExtension() string {
	format := strings.ToLower(p.config.VideoOutputFormat)
	switch format {
	case "mp4":
		return ".mp4"
	case "avi":
		return ".avi"
	case "mkv":
		return ".mkv"
	case "webm":
		return ".webm"
	case "mov":
		return ".mov"
	default:
		// Default to mp4 if unknown format
		return ".mp4"
	}
}

// GetCaptureFileExtension returns the file extension for the configured capture codec
func (p *VideoProcessor) GetCaptureFileExtension() string {
	codec := strings.ToUpper(p.config.CaptureCodec)
	switch codec {
	case "MJPG":
		return ".avi" // MJPG is typically stored in AVI containers
	case "MP4V":
		return ".mp4"
	case "YUYV":
		return ".avi" // Raw formats typically use AVI
	case "H264":
		return ".mp4"
	default:
		// Default to avi for most capture codecs
		return ".avi"
	}
}

// GetVideoDuration extracts the actual duration from video metadata using goffmpeg's efficient probing
// This uses goffmpeg's internal ffprobe functionality which is much more efficient than exec calls
func (p *VideoProcessor) GetVideoDuration(videoPath string) (time.Duration, error) {
	// Create transcoder instance for metadata probing only
	trans := new(transcoder.Transcoder)

	// Initialize transcoder - this runs ffprobe internally to get metadata
	err := trans.Initialize(videoPath, "") // Empty output path since we're just probing
	if err != nil {
		return 0, fmt.Errorf("failed to probe video metadata: %w", err)
	}

	// Get the duration from the already-parsed metadata
	// The Initialize method runs ffprobe and stores the result in MediaFile metadata
	durationStr := trans.MediaFile().Metadata().Format.Duration
	if durationStr == "" {
		return 0, fmt.Errorf("empty duration in video metadata")
	}

	// Parse duration string to float64 seconds
	durationSeconds, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %w", durationStr, err)
	}

	if durationSeconds <= 0 {
		return 0, fmt.Errorf("invalid or zero duration: %f seconds", durationSeconds)
	}

	// Convert to time.Duration
	duration := time.Duration(durationSeconds * float64(time.Second))
	return duration, nil
}

// DetectMotion analyzes a video file for motion using gocv. This implementation is more robust
// against noise and lighting changes by using blurring, thresholding, and contour analysis.
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

	gray := gocv.NewMat()
	defer gray.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()

	fgMask := gocv.NewMat()
	defer fgMask.Close()

	thresh := gocv.NewMat()
	defer thresh.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	motionDetected := false
	frameCount := 0
	maxFramesToCheck := 150 // Check more frames for better background establishment

	// Minimum contour area to be considered motion. This can be tuned.
	// It replaces the less reliable percentage-based sensitivity.
	minArea := p.config.MotionMinArea
	if minArea <= 0 {
		minArea = 500 // Default value if not configured
	}

	for frameCount < maxFramesToCheck {
		if ok := video.Read(&img); !ok {
			break // End of video
		}

		if img.Empty() {
			continue
		}

		// 1. Convert to grayscale
		gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

		// 2. Apply Gaussian blur to smooth the image and reduce noise
		gocv.GaussianBlur(gray, &blurred, image.Pt(21, 21), 0, 0, gocv.BorderDefault)

		// 3. Apply background subtraction
		detector.Apply(blurred, &fgMask)

		// 4. Threshold the foreground mask to get a binary image
		gocv.Threshold(fgMask, &thresh, 25, 255, gocv.ThresholdBinary)

		// 5. Dilate the thresholded image to fill in holes
		gocv.Dilate(thresh, &thresh, kernel)

		// 6. Find contours of the moving objects
		contours := gocv.FindContours(thresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)

		// 7. Check if any contour is large enough
		for i := range contours.Size() {
			area := gocv.ContourArea(contours.At(i))
			if area > float64(minArea) {
				motionDetected = true
				break
			}
		}
		contours.Close()

		if motionDetected {
			break
		}

		frameCount++
	}

	return motionDetected, nil
}

// StartContinuousCapture starts continuous video capture with chunking
func (p *VideoProcessor) StartContinuousCapture(device string, chunkDuration time.Duration, onChunkReady func(*models.VideoClip)) error {
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
func (p *VideoProcessor) captureLoop(chunkDuration time.Duration, onChunkReady func(*models.VideoClip)) {
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
		// Create unique filename for this chunk using the appropriate extension for the capture codec
		chunkCount++
		captureExtension := p.GetCaptureFileExtension()
		chunkFile := fmt.Sprintf("%s/chunk_%d_%d%s", p.tempDir, time.Now().Unix(), chunkCount, captureExtension)

		// Record chunk
		recordingStartTime, err := p.recordChunk(chunkFile, chunkDuration)
		if err != nil {
			fmt.Printf("Error recording chunk: %v\n", err)
			// Don't continue immediately, check for stop signal first
		} else {
			// Notify that chunk is ready only if recording was successful
			if onChunkReady != nil {
				videoClip := &models.VideoClip{
					Filename:  chunkFile,
					Timestamp: recordingStartTime,
					Duration:  chunkDuration,
					HasMotion: false, // Will be determined later during processing
					FilePath:  chunkFile,
				}
				onChunkReady(videoClip)
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
func (p *VideoProcessor) recordChunk(outputPath string, duration time.Duration) (time.Time, error) {
	if p.webcam == nil {
		return time.Time{}, fmt.Errorf("webcam not initialized")
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
		return time.Time{}, fmt.Errorf("failed to create video writer: %w", err)
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
		return time.Time{}, fmt.Errorf("no frames were recorded from webcam")
	}

	return startTime, nil
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

// CleanupTempFiles removes temporary files older than maxAge
func (p *VideoProcessor) CleanupTempFiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(p.tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	cutoffTime := time.Now().Add(-maxAge)
	var cleanedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			fmt.Printf("Warning: failed to get file info for %s: %v\n", entry.Name(), err)
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			filePath := fmt.Sprintf("%s/%s", p.tempDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("Warning: failed to remove temp file %s: %v\n", filePath, err)
			} else {
				fmt.Printf("Cleaned up temp file: %s\n", filePath)
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		fmt.Printf("Cleaned up %d temporary files\n", cleanedCount)
	}

	return nil
}
