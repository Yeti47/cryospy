package videos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xfrr/goffmpeg/transcoder"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

// VideoMetadataExtractor defines the interface for extracting video metadata
type VideoMetadataExtractor interface {
	// ExtractMetadata extracts metadata from video data
	ExtractMetadata(videoData []byte) (*VideoMetadata, error)
}

// FFmpegMetadataExtractor implements VideoMetadataExtractor using FFmpeg
type FFmpegMetadataExtractor struct {
	logger logging.Logger
}

// NewFFmpegMetadataExtractor creates a new FFmpeg-based metadata extractor
func NewFFmpegMetadataExtractor(logger logging.Logger) *FFmpegMetadataExtractor {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &FFmpegMetadataExtractor{
		logger: logger,
	}
}

// ExtractMetadata extracts metadata from video data using goffmpeg
func (e *FFmpegMetadataExtractor) ExtractMetadata(videoData []byte) (*VideoMetadata, error) {
	// Create temporary file for metadata extraction
	tempDir, err := os.MkdirTemp("", "video_metadata_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write video to temporary file with generic extension
	videoFile := filepath.Join(tempDir, "input.mp4") // Default to mp4, ffmpeg will handle it
	err = os.WriteFile(videoFile, videoData, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write video file: %w", err)
	}

	// Create transcoder to probe metadata
	trans := new(transcoder.Transcoder)
	err = trans.Initialize(videoFile, "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transcoder for metadata: %w", err)
	}

	// Get video metadata
	metadata := trans.MediaFile().Metadata()

	// Extract video stream information
	var width, height int
	var mimeType, extension string

	for _, stream := range metadata.Streams {
		if stream.CodecType == "video" {
			width = stream.Width
			height = stream.Height

			// Determine MIME type and extension based on codec
			switch stream.CodecName {
			case "h264":
				mimeType = "video/mp4"
				extension = "mp4"
			case "h265", "hevc":
				mimeType = "video/mp4"
				extension = "mp4"
			case "vp8", "vp9":
				mimeType = "video/webm"
				extension = "webm"
			case "av1":
				mimeType = "video/webm"
				extension = "webm"
			default:
				// Fallback based on container format
				if metadata.Format.FormatName != "" {
					if strings.Contains(metadata.Format.FormatName, "mp4") {
						mimeType = "video/mp4"
						extension = "mp4"
					} else if strings.Contains(metadata.Format.FormatName, "webm") {
						mimeType = "video/webm"
						extension = "webm"
					} else if strings.Contains(metadata.Format.FormatName, "avi") {
						mimeType = "video/avi"
						extension = "avi"
					} else {
						mimeType = "video/mp4" // Default fallback
						extension = "mp4"
					}
				} else {
					mimeType = "video/mp4" // Default fallback
					extension = "mp4"
				}
			}
			break // Use first video stream
		}
	}

	if width == 0 || height == 0 {
		return nil, fmt.Errorf("could not extract video dimensions")
	}

	e.logger.Debug(fmt.Sprintf("Extracted video metadata: %dx%d, %s", width, height, mimeType))

	return &VideoMetadata{
		Width:     width,
		Height:    height,
		MimeType:  mimeType,
		Extension: extension,
	}, nil
}
