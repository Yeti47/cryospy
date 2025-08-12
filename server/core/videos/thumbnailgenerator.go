package videos

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xfrr/goffmpeg/transcoder"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

// ThumbnailGenerator defines the interface for generating video thumbnails
type ThumbnailGenerator interface {
	// GenerateThumbnail generates a thumbnail from video data
	GenerateThumbnail(videoData []byte, videoMeta *VideoMetadata) (*Thumbnail, error)
}

// FFmpegThumbnailGenerator implements ThumbnailGenerator using FFmpeg
type FFmpegThumbnailGenerator struct {
	logger logging.Logger
}

// NewFFmpegThumbnailGenerator creates a new FFmpeg-based thumbnail generator
func NewFFmpegThumbnailGenerator(logger logging.Logger) *FFmpegThumbnailGenerator {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &FFmpegThumbnailGenerator{
		logger: logger,
	}
}

// calculateThumbnailDimensions calculates optimal thumbnail dimensions preserving aspect ratio
func (g *FFmpegThumbnailGenerator) calculateThumbnailDimensions(videoWidth, videoHeight int) (int, int) {
	const maxThumbnailWidth = 480
	const maxThumbnailHeight = 360

	// Calculate aspect ratio
	aspectRatio := float64(videoWidth) / float64(videoHeight)

	var thumbWidth, thumbHeight int

	// Scale based on which dimension hits the limit first
	if float64(maxThumbnailWidth)/aspectRatio <= float64(maxThumbnailHeight) {
		// Width-constrained
		thumbWidth = maxThumbnailWidth
		thumbHeight = int(float64(maxThumbnailWidth) / aspectRatio)
	} else {
		// Height-constrained
		thumbHeight = maxThumbnailHeight
		thumbWidth = int(float64(maxThumbnailHeight) * aspectRatio)
	}

	// Ensure even dimensions (some codecs prefer this)
	thumbWidth = (thumbWidth / 2) * 2
	thumbHeight = (thumbHeight / 2) * 2

	g.logger.Debug(fmt.Sprintf("Calculated thumbnail dimensions: %dx%d (from %dx%d)",
		thumbWidth, thumbHeight, videoWidth, videoHeight))

	return thumbWidth, thumbHeight
}

// GenerateThumbnail generates a thumbnail from video data using goffmpeg
func (g *FFmpegThumbnailGenerator) GenerateThumbnail(videoData []byte, videoMeta *VideoMetadata) (*Thumbnail, error) {
	// Create temporary directory for processing
	tempDir, err := os.MkdirTemp("", "video_thumbnail_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write video data to temporary file
	videoFile := filepath.Join(tempDir, fmt.Sprintf("input.%s", videoMeta.Extension))
	err = os.WriteFile(videoFile, videoData, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write video file: %w", err)
	}

	// Output thumbnail file
	thumbnailFile := filepath.Join(tempDir, "thumbnail.png")

	// Calculate optimal thumbnail dimensions
	thumbWidth, thumbHeight := g.calculateThumbnailDimensions(videoMeta.Width, videoMeta.Height)

	// Create transcoder instance
	trans := new(transcoder.Transcoder)

	// Initialize transcoder with input file
	err = trans.Initialize(videoFile, thumbnailFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transcoder: %w", err)
	}

	// Set transcoder options for thumbnail extraction with calculated dimensions
	trans.MediaFile().SetSeekTime("00:00:01")                                             // Seek to 1 second
	trans.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:%d", thumbWidth, thumbHeight)) // Use calculated dimensions
	trans.MediaFile().SetVideoCodec("png")                                                // Output as PNG
	trans.MediaFile().SetSkipAudio(true)                                                  // Skip audio processing
	trans.MediaFile().SetOutputFormat("image2")                                           // Single image output
	trans.MediaFile().SetVideoBitRate("1")                                                // Minimal bitrate for single frame

	// Process the video to extract thumbnail
	done := trans.Run(true) // true = progress reporting
	err = <-done
	if err != nil {
		return nil, fmt.Errorf("ffmpeg transcoding failed: %w", err)
	}

	// Read the generated thumbnail
	thumbnailData, err := os.ReadFile(thumbnailFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read thumbnail file: %w", err)
	}

	g.logger.Debug(fmt.Sprintf("Generated thumbnail: %dx%d PNG, %d bytes",
		thumbWidth, thumbHeight, len(thumbnailData)))

	return &Thumbnail{
		Data:     thumbnailData,
		Width:    thumbWidth,
		Height:   thumbHeight,
		MimeType: "image/png",
	}, nil
}
