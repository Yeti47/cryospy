package common

import (
	"fmt"
	"log"
	"maps"
	"os/exec"
	"regexp"
	"strings"
)

// CodecFallbackMap defines fallback chains for video codecs
var CodecFallbackMap = map[string][]string{
	// H.264 codecs in preference order
	"libx264":      {"libx264", "libopenh264", "h264_vaapi", "h264_qsv", "h264_v4l2m2m"},
	"libopenh264":  {"libopenh264", "libx264", "h264_vaapi", "h264_qsv", "h264_v4l2m2m"},
	"h264_vaapi":   {"h264_vaapi", "libx264", "libopenh264", "h264_qsv", "h264_v4l2m2m"},
	"h264_qsv":     {"h264_qsv", "libx264", "libopenh264", "h264_vaapi", "h264_v4l2m2m"},
	"h264_v4l2m2m": {"h264_v4l2m2m", "libx264", "libopenh264", "h264_vaapi", "h264_qsv"},

	// H.265 falls back to H.264 codecs
	"libx265": {"libx265", "libx264", "libopenh264", "h264_vaapi", "h264_qsv", "h264_v4l2m2m"},

	// Add more codec families as needed
}

// CodecProvider interface for managing codec availability and fallbacks
type CodecProvider interface {
	IsCodecAvailable(codec string) bool
	GetFallbackCodec(requestedCodec string) (string, error)
	GetAvailableCodecs() map[string]bool
}

// FFmpegCodecProvider implements CodecProvider using FFmpeg
type FFmpegCodecProvider struct {
	// Cache to avoid repeated FFmpeg calls
	availableCodecs map[string]bool
}

// NewFFmpegCodecProvider creates a new FFmpeg-based codec provider
func NewFFmpegCodecProvider() *FFmpegCodecProvider {
	provider := &FFmpegCodecProvider{
		availableCodecs: make(map[string]bool),
	}

	// Load available codecs immediately
	provider.loadAvailableCodecs()

	return provider
}

// IsCodecAvailable checks if a codec is available by querying FFmpeg
func (c *FFmpegCodecProvider) IsCodecAvailable(codec string) bool {
	available, exists := c.availableCodecs[codec]
	return exists && available
}

// GetAvailableCodecs returns a copy of all available codecs
func (c *FFmpegCodecProvider) GetAvailableCodecs() map[string]bool {
	// Return a copy to prevent external modification
	result := make(map[string]bool)
	maps.Copy(result, c.availableCodecs)
	return result
}

// loadAvailableCodecs queries FFmpeg for all available encoders and caches the result
func (c *FFmpegCodecProvider) loadAvailableCodecs() {
	cmd := exec.Command("ffmpeg", "-encoders")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to query FFmpeg encoders: %v", err)
		// Fallback to hardcoded known unavailable codecs
		c.availableCodecs["libx264"] = false
		return
	}

	// Parse FFmpeg encoder output using regex
	// Pattern matches lines like: " V....D libopenh264          OpenH264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)"
	// Captures: flags (V..... or A.....) and codec name
	// Excludes header lines that have " = " in them
	codecPattern := regexp.MustCompile(`^ ([VA][.SFXBD]{5})\s+([a-zA-Z0-9_-]+)\s+`)

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		// Skip header lines that contain " = "
		if strings.Contains(line, " = ") {
			continue
		}

		matches := codecPattern.FindStringSubmatch(line)
		if len(matches) >= 3 {
			flags := matches[1]
			codecName := matches[2]

			// Ensure we have a valid codec name and it's a video/audio encoder
			if len(codecName) > 0 && (strings.HasPrefix(flags, "V") || strings.HasPrefix(flags, "A")) {
				c.availableCodecs[codecName] = true
			}
		}
	}

	log.Printf("Loaded %d available codecs from FFmpeg", len(c.availableCodecs))
}

// GetFallbackCodec finds the first available codec from the fallback chain
func (c *FFmpegCodecProvider) GetFallbackCodec(requestedCodec string) (string, error) {
	// Check if the requested codec is available first
	if c.IsCodecAvailable(requestedCodec) {
		return requestedCodec, nil
	}

	// Look up fallback chain
	fallbackChain, exists := CodecFallbackMap[requestedCodec]
	if !exists {
		// No fallback defined for this codec
		return "", fmt.Errorf("codec '%s' is not available and no fallback is defined", requestedCodec)
	}

	log.Printf("Codec '%s' not available, trying fallbacks: %v", requestedCodec, fallbackChain)

	// Try each codec in the fallback chain
	for _, codec := range fallbackChain {
		if c.IsCodecAvailable(codec) {
			log.Printf("Using fallback codec: %s", codec)
			return codec, nil
		}
	}

	return "", fmt.Errorf("no suitable codec available from fallback chain: %v", fallbackChain)
}
