package streaming

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xfrr/goffmpeg/transcoder"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
)

// ClipNormalizer interface for normalizing video clips for HLS streaming
type ClipNormalizer interface {
	// NormalizeClip transcodes a DecryptedClip to a format suitable for HLS streaming
	// Returns the normalized video data as a .ts segment and any error
	NormalizeClip(clip *videos.DecryptedClip) ([]byte, error)
}

// FFmpegClipNormalizer implements ClipNormalizer using goffmpeg
type FFmpegClipNormalizer struct {
	logger   logging.Logger
	tempDir  string
	settings NormalizationSettings
}

// NormalizationSettings contains configuration for video normalization
type NormalizationSettings struct {
	// Target resolution (width x height)
	Width  int
	Height int
	// Target bitrate in kbps
	VideoBitrate string
	// Video codec
	VideoCodec string
	// Frame rate
	FrameRate int
}

// DefaultNormalizationSettings returns sensible defaults for 480p HLS streaming
func DefaultNormalizationSettings() NormalizationSettings {
	return NormalizationSettings{
		Width:        854,       // 480p width (16:9 aspect ratio)
		Height:       480,       // 480p height
		VideoBitrate: "1000k",   // 1 Mbps for decent quality at 480p
		VideoCodec:   "libx264", // H.264 for compatibility
		FrameRate:    25,        // 25 fps for efficient streaming
	}
}

// NewFFmpegClipNormalizer creates a new FFmpeg-based clip normalizer
func NewFFmpegClipNormalizer(logger logging.Logger, tempDir string) *FFmpegClipNormalizer {
	if logger == nil {
		logger = logging.NopLogger
	}

	if tempDir == "" {
		tempDir = os.TempDir()
	}

	return &FFmpegClipNormalizer{
		logger:   logger,
		tempDir:  tempDir,
		settings: DefaultNormalizationSettings(),
	}
}

// NewFFmpegClipNormalizerWithSettings creates a new FFmpeg-based clip normalizer with custom settings
func NewFFmpegClipNormalizerWithSettings(logger logging.Logger, tempDir string, settings NormalizationSettings) *FFmpegClipNormalizer {
	if logger == nil {
		logger = logging.NopLogger
	}

	if tempDir == "" {
		tempDir = os.TempDir()
	}

	return &FFmpegClipNormalizer{
		logger:   logger,
		tempDir:  tempDir,
		settings: settings,
	}
}

// NormalizeClip transcodes the given DecryptedClip to a .ts segment optimized for HLS streaming
func (n *FFmpegClipNormalizer) NormalizeClip(clip *videos.DecryptedClip) ([]byte, error) {
	// Validate input clip
	if clip == nil {
		return nil, fmt.Errorf("clip cannot be nil")
	}
	if len(clip.Video) == 0 {
		return nil, fmt.Errorf("clip video data cannot be empty")
	}
	if clip.ID == "" {
		return nil, fmt.Errorf("clip ID cannot be empty")
	}

	n.logger.Info("Starting clip normalization", "clipID", clip.ID, "originalSize", len(clip.Video))

	// Create temporary files for processing
	timestamp := time.Now().UnixNano()
	inputFile := filepath.Join(n.tempDir, fmt.Sprintf("input_%d_%s.tmp", timestamp, clip.ID))
	outputFile := filepath.Join(n.tempDir, fmt.Sprintf("output_%d_%s.ts", timestamp, clip.ID))

	// Ensure cleanup of input file
	defer os.Remove(inputFile)

	// Write the decrypted video data to a temporary file
	if err := os.WriteFile(inputFile, clip.Video, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input video to temp file: %w", err)
	}

	// Create transcoder instance
	trans := new(transcoder.Transcoder)

	// Initialize transcoder with input file
	err := trans.Initialize(inputFile, outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transcoder: %w", err)
	}

	// Configure transcoder options for HLS-optimized .ts output (video only)
	trans.MediaFile().SetVideoCodec(n.settings.VideoCodec)
	trans.MediaFile().SetVideoBitRate(n.settings.VideoBitrate)
	trans.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:%d", n.settings.Width, n.settings.Height))
	trans.MediaFile().SetFrameRate(n.settings.FrameRate)

	// Disable audio - videos don't have audio
	trans.MediaFile().SetSkipAudio(true)

	// Add keyframe interval for better seeking (every 2 seconds at 25fps = 50 frames)
	trans.MediaFile().SetKeyframeInterval(50)

	n.logger.Info("Starting transcoding", "clipID", clip.ID,
		"targetResolution", fmt.Sprintf("%dx%d", n.settings.Width, n.settings.Height),
		"videoBitrate", n.settings.VideoBitrate)

	// Start transcoding process
	done := trans.Run(false) // false = not streaming mode
	err = <-done
	if err != nil {
		return nil, fmt.Errorf("transcoding failed: %w", err)
	}

	// Read the transcoded .ts file
	normalizedData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcoded output: %w", err)
	}

	// Clean up the output file after reading
	os.Remove(outputFile)

	n.logger.Info("Clip normalization completed", "clipID", clip.ID,
		"originalSize", len(clip.Video),
		"normalizedSize", len(normalizedData),
		"compressionRatio", fmt.Sprintf("%.2f", float64(len(clip.Video))/float64(len(normalizedData))))

	return normalizedData, nil
}
