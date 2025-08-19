package recording

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/common"
	"github.com/yeti47/cryospy/client/capture-client/config"
	"gocv.io/x/gocv"
)

var DefaultRecordingSettings = RecordingSettings{
	ClipDuration: 30 * time.Second, // Default clip duration of 10 seconds
	Codec:        "MJPG",           // Default codec
	FrameRate:    15.0,             // Default frame rate of 30 FPS
}

type RecordingCallback func(clip *RawClip) error
type RecordingErrorCallback func(err error) (cancel bool)

type Recorder interface {
	// StartRecording begins a new recording session
	// The callback will be called with each recorded clip
	StartRecording(callback RecordingCallback, errorCallback RecordingErrorCallback) (bool, error)
	// StopRecording ends the current recording session
	StopRecording() error
	// IsRecording checks if a recording is currently in progress
	IsRecording() bool
}

type GoCVRecorder struct {
	device           string // Device identifier, e.g., "/dev/video0" or "0" for default camera
	clipDirectory    string
	settingsProvider config.SettingsProvider[RecordingSettings]
	isRecording      bool
	mu               sync.RWMutex
}

func NewGoCVRecorder(device string, clipDirectory string, provider config.SettingsProvider[RecordingSettings]) *GoCVRecorder {
	return &GoCVRecorder{
		device:           device,
		clipDirectory:    clipDirectory,
		settingsProvider: provider,
		isRecording:      false,
	}
}

func (r *GoCVRecorder) IsRecording() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRecording
}

func (r *GoCVRecorder) StartRecording(callback RecordingCallback, errorCallback RecordingErrorCallback) (bool, error) {
	r.mu.Lock()
	if r.isRecording {
		r.mu.Unlock()
		return false, nil // Already recording
	}
	r.isRecording = true
	r.mu.Unlock()

	// Parse device ID
	deviceID := 0
	if r.device != "" && r.device != "0" {
		if id, err := strconv.Atoi(r.device); err == nil {
			deviceID = id
		}
	}

	// Open webcam
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		r.mu.Lock()
		r.isRecording = false
		r.mu.Unlock()
		return false, fmt.Errorf("failed to open webcam: %w", err)
	}

	// Start a recording goroutine and pass the webcam instance to it.
	// The goroutine is now responsible for the entire lifecycle of the webcam.
	go func(webcam *gocv.VideoCapture) {
		// This defer ensures that resources are cleaned up safely by this goroutine
		defer func() {
			r.mu.Lock()
			if webcam != nil {
				log.Println("Closing webcam from recording loop...")
				webcam.Close()
			}
			r.isRecording = false
			r.mu.Unlock()
		}()

		for clipIndex := 0; ; clipIndex++ {
			r.mu.RLock()
			if !r.isRecording {
				r.mu.RUnlock()
				break // Exit loop if recording has been stopped
			}
			r.mu.RUnlock()

			// Get the latest settings for this clip from the provider.
			// The provider is responsible for its own thread safety.
			settingsSnapshot := r.settingsProvider.GetSettings()

			clip, err := r.recordNextClip(webcam, clipIndex, settingsSnapshot)
			if err != nil {
				log.Printf("Error recording clip %d: %v", clipIndex, err)
				if errorCallback != nil {
					cancel := errorCallback(err) // Call error callback if provided
					if cancel {
						break
					}
				}
				continue // Skip to next clip
			}

			// Call the callback with the recorded clip
			if err := callback(clip); err != nil {
				log.Printf("Error in callback for clip %d: %v", clipIndex, err)

				if errorCallback != nil {
					// Create a new error to provide more context
					callbackErr := fmt.Errorf("processing callback failed for clip %s: %w", clip.Path, err)
					if cancel := errorCallback(callbackErr); cancel {
						break
					}
				}
			}
		}

	}(webcam)

	return true, nil
}

func (r *GoCVRecorder) recordNextClip(webcam *gocv.VideoCapture, clipIndex int, settingsSnapshot RecordingSettings) (*RawClip, error) {

	if webcam == nil {
		return nil, fmt.Errorf("webcam not initialized")
	}

	fileExtension := common.CodecToFileExtension(settingsSnapshot.Codec)

	clipPath := fmt.Sprintf("%s/clip_%d_%d%s", r.clipDirectory, time.Now().Unix(), clipIndex, fileExtension)

	log.Printf("Recording clip %d to %s with codec %s", clipIndex, clipPath, settingsSnapshot.Codec)

	// Get frame properties from webcam
	width := int(webcam.Get(gocv.VideoCaptureFrameWidth))
	height := int(webcam.Get(gocv.VideoCaptureFrameHeight))

	if width <= 0 || height <= 0 {
		width = 640
		height = 480
		log.Printf("Using default resolution: %dx%d", width, height)
	} else {
		log.Printf("Using webcam resolution: %dx%d", width, height)
	}

	// Create video writer - use configured codec
	writer, err := gocv.VideoWriterFile(clipPath, settingsSnapshot.Codec, settingsSnapshot.FrameRate, width, height, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create video writer: %w", err)
	}
	defer func() {
		log.Printf("Closing video writer for %s", clipPath)
		writer.Close()
	}()

	img := gocv.NewMat()
	defer img.Close()

	startTime := time.Now()
	frameCount := 0

	// Calculate frame interval for precise timing control
	frameInterval := time.Duration(float64(time.Second) / settingsSnapshot.FrameRate)
	nextFrameTime := startTime

	for time.Since(startTime) < settingsSnapshot.ClipDuration {
		r.mu.RLock()
		isRec := r.isRecording
		r.mu.RUnlock()

		if !isRec {
			break
		}

		// Wait until it's time for the next frame to maintain proper frame rate
		now := time.Now()
		if now.Before(nextFrameTime) {
			time.Sleep(nextFrameTime.Sub(now))
		}

		if ok := webcam.Read(&img); !ok {
			log.Printf("Failed to read frame %d from webcam", frameCount)
			time.Sleep(time.Millisecond * 67) // Wait a bit before retrying
			// Don't advance nextFrameTime on failed reads
			continue
		}

		if img.Empty() {
			log.Printf("Empty frame %d from webcam", frameCount)
			// Don't advance nextFrameTime on empty frames
			continue
		}

		// Write frame to video
		if err := writer.Write(img); err != nil {
			log.Printf("Failed to write frame %d to video: %v", frameCount, err)
		}
		frameCount++

		// Calculate the time for the next frame
		nextFrameTime = nextFrameTime.Add(frameInterval)
	}

	clipDuration := time.Since(startTime)

	log.Printf("Recorded %d frames for %s in %v", frameCount, clipPath, clipDuration)

	// Check if we recorded any frames
	if frameCount <= 0 {
		return nil, fmt.Errorf("no frames were recorded from webcam")
	}

	clip := &RawClip{
		Path:      clipPath,
		Codec:     settingsSnapshot.Codec,
		Timestamp: startTime.UTC(),
		Duration:  clipDuration,
		Frames:    frameCount,
		FrameRate: settingsSnapshot.FrameRate,
	}

	return clip, nil

}

func (r *GoCVRecorder) StopRecording() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.isRecording = false
	// The recording goroutine will handle closing and cleaning up the webcam.
	return nil
}
