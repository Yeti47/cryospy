package postprocessing

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/xfrr/goffmpeg/transcoder"
	"github.com/yeti47/cryospy/client/capture-client/common"
	"github.com/yeti47/cryospy/client/capture-client/config"
	"github.com/yeti47/cryospy/client/capture-client/recording"
)

type PostProcessor interface {
	// ProcessVideo processes a raw video clip and returns a processed video clip.
	ProcessVideo(rawClip *recording.RawClip) (*VideoClip, error)
}

type FfmpegPostProcessor struct {
	settingsProvider config.SettingsProvider[PostProcessingSettings]
	codecProvider    common.CodecProvider
}

func NewFfmpegPostProcessor(settingsProvider config.SettingsProvider[PostProcessingSettings], codecProvider common.CodecProvider) *FfmpegPostProcessor {
	return &FfmpegPostProcessor{
		settingsProvider: settingsProvider,
		codecProvider:    codecProvider,
	}
}

func (p *FfmpegPostProcessor) ProcessVideo(rawClip *recording.RawClip) (*VideoClip, error) {
	startTime := time.Now()

	log.Printf("Starting video post-processing - Input: '%s', Duration: %v", rawClip.Path, rawClip.Duration)

	// Get the latest settings for this operation.
	// The provider is responsible for its own thread safety.
	settings := p.settingsProvider.GetSettings()

	// Log post-processing settings for debugging
	log.Printf("Post-processing settings - Format: '%s', Codec: '%s', BitRate: '%s', Grayscale: %v, Resolution: '%s'",
		settings.OutputFormat, settings.OutputCodec, settings.VideoBitRate, settings.Grayscale, settings.DownscaleResolution.Format("w:h"))

	// Validate and apply codec fallback if necessary
	codec, err := p.codecProvider.GetFallbackCodec(settings.OutputCodec)
	if err != nil {
		log.Printf("Codec validation failed for '%s': %v", settings.OutputCodec, err)
		return nil, fmt.Errorf("codec validation failed: %w", err)
	}

	if codec != settings.OutputCodec {
		log.Printf("Codec fallback applied: '%s' -> '%s'", settings.OutputCodec, codec)
	}

	// Create transcoder instance
	trans := new(transcoder.Transcoder)

	outputPath := p.getOutputPath(rawClip, settings)
	log.Printf("Output path: '%s'", outputPath)

	// Initialize transcoder with input and output files
	log.Printf("Initializing transcoder with codec '%s'", codec)
	err = trans.Initialize(rawClip.Path, outputPath)
	if err != nil {
		log.Printf("Failed to initialize transcoder: %v", err)
		return nil, fmt.Errorf("failed to initialize transcoder: %w", err)
	}

	// Configure basic output settings - video only, no audio
	trans.MediaFile().SetVideoCodec(codec)
	trans.MediaFile().SetOutputFormat(settings.OutputFormat)

	// Disable audio streams
	trans.MediaFile().SetSkipAudio(true)

	// Build video filters
	var filters []string

	// Apply grayscale filter if enabled
	if settings.Grayscale {
		// Use different grayscale methods based on codec compatibility
		if codec == "libx264" {
			// For H.264, use saturation-based grayscale that maintains YUV420P format
			filters = append(filters, "hue=s=0")
			log.Printf("Applied H.264-compatible grayscale filter (desaturation)")
		} else {
			// For other codecs (like H.265), use the standard gray format
			filters = append(filters, "format=gray")
			log.Printf("Applied standard grayscale filter")
		}
	}

	// Apply downscaling if specified
	if !settings.DownscaleResolution.IsEmpty() {
		formattedResolution := settings.DownscaleResolution.Format("w:h")
		filters = append(filters, fmt.Sprintf("scale=%s", formattedResolution))
		log.Printf("Applied downscaling filter: %s", formattedResolution)
	}

	// Apply filters if any
	if len(filters) > 0 {
		filterChain := strings.Join(filters, ",")
		trans.MediaFile().SetVideoFilter(filterChain)
		log.Printf("Applied video filters: %s", filterChain)
	} else {
		log.Printf("No video filters applied")
	}

	// Apply compression settings for smaller file size (video only)
	trans.MediaFile().SetVideoBitRate(settings.VideoBitRate) // Use configured bitrate
	log.Printf("Set video bitrate: %s", settings.VideoBitRate)

	// Apply codec-specific optimizations for faster processing
	if codec == "libx264" {
		trans.MediaFile().SetPreset("ultrafast")
		trans.MediaFile().SetVideoProfile("baseline")
		log.Printf("Applied H.264 optimizations: preset=ultrafast profile=baseline")
	}

	// Run the transcoding
	// Disable progress reporting on Windows due to pipe handling issues
	enableProgress := runtime.GOOS != "windows"
	log.Printf("Starting transcoding (progress reporting: %v, OS: %s)", enableProgress, runtime.GOOS)
	done := trans.Run(enableProgress)

	// Get the duration from the input file's metadata, which was already probed during initialization.
	// This avoids spawning a second ffprobe process.
	duration, err := p.parseDuration(trans.MediaFile().Metadata().Format.Duration)
	if err != nil {
		log.Printf("Failed to parse duration from metadata (%s), using fallback: %v", trans.MediaFile().Metadata().Format.Duration, rawClip.Duration)
		// Fallback to the duration from the raw clip struct if parsing fails
		duration = rawClip.Duration
	} else {
		log.Printf("Parsed duration from metadata: %v", duration)
	}

	log.Printf("Waiting for transcoding to complete...")
	err = <-done
	transcodeEnd := time.Now()
	processingTime := transcodeEnd.Sub(startTime)

	if err != nil {
		log.Printf("Transcoding failed after %v: %v", processingTime, err)
		return nil, fmt.Errorf("failed to process video: %w", err)
	}

	log.Printf("Transcoding completed successfully in %v", processingTime)

	result := &VideoClip{
		Path:      outputPath,
		Codec:     settings.OutputCodec,
		Format:    settings.OutputFormat,
		Timestamp: rawClip.Timestamp,
		Duration:  duration,
	}

	log.Printf("Post-processing completed - Output: '%s', Final duration: %v, Total time: %v",
		result.Path, result.Duration, processingTime)

	return result, nil
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
