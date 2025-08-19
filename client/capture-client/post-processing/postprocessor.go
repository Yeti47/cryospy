package postprocessing

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xfrr/goffmpeg/transcoder"
	"github.com/yeti47/cryospy/client/capture-client/config"
	"github.com/yeti47/cryospy/client/capture-client/recording"
)

type PostProcessor interface {
	// ProcessVideo processes a raw video clip and returns a processed video clip.
	ProcessVideo(rawClip *recording.RawClip) (*VideoClip, error)
}

type FfmpegPostProcessor struct {
	settingsProvider config.SettingsProvider[PostProcessingSettings]
}

func NewFfmpegPostProcessor(settingsProvider config.SettingsProvider[PostProcessingSettings]) *FfmpegPostProcessor {
	return &FfmpegPostProcessor{
		settingsProvider: settingsProvider,
	}
}

func (p *FfmpegPostProcessor) ProcessVideo(rawClip *recording.RawClip) (*VideoClip, error) {

	// Get the latest settings for this operation.
	// The provider is responsible for its own thread safety.
	settings := p.settingsProvider.GetSettings()

	// Create transcoder instance
	trans := new(transcoder.Transcoder)

	outputPath := p.getOutputPath(rawClip, settings)

	// Initialize transcoder with input and output files
	err := trans.Initialize(rawClip.Path, outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transcoder: %w", err)
	}

	// Configure basic output settings - video only, no audio
	trans.MediaFile().SetVideoCodec(settings.OutputCodec)
	trans.MediaFile().SetOutputFormat(settings.OutputFormat)

	// Disable audio streams
	trans.MediaFile().SetSkipAudio(true)

	// Build video filters
	var filters []string

	// Apply grayscale filter if enabled
	if settings.Grayscale {
		filters = append(filters, "format=gray")
	}

	// Apply downscaling if specified
	if !settings.DownscaleResolution.IsEmpty() {
		formattedResolution := settings.DownscaleResolution.Format("w:h")
		filters = append(filters, fmt.Sprintf("scale=%s", formattedResolution))
	}

	// Apply filters if any
	if len(filters) > 0 {
		filterChain := strings.Join(filters, ",")
		trans.MediaFile().SetVideoFilter(filterChain)
	}

	// Apply compression settings for smaller file size (video only)
	trans.MediaFile().SetVideoBitRate(settings.VideoBitRate) // Use configured bitrate

	// Run the transcoding
	done := trans.Run(true) // true = progress reporting

	// Get the duration from the input file's metadata, which was already probed during initialization.
	// This avoids spawning a second ffprobe process.
	duration, err := p.parseDuration(trans.MediaFile().Metadata().Format.Duration)
	if err != nil {
		// Fallback to the duration from the raw clip struct if parsing fails
		duration = rawClip.Duration
	}

	err = <-done

	if err != nil {
		return nil, fmt.Errorf("failed to process video: %w", err)
	}

	return &VideoClip{
		Path:      outputPath,
		Codec:     settings.OutputCodec,
		Format:    settings.OutputFormat,
		Timestamp: rawClip.Timestamp,
		Duration:  duration,
	}, nil
}

func (p *FfmpegPostProcessor) getOutputPath(rawClip *recording.RawClip, settings PostProcessingSettings) string {

	// Generate output path based on raw clip path and configured output format
	outputPath := strings.TrimSuffix(rawClip.Path, filepath.Ext(rawClip.Path)) + "." + strings.TrimLeft(settings.OutputFormat, ".")
	return outputPath
}

func (p *FfmpegPostProcessor) parseDuration(durationStr string) (time.Duration, error) {
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
