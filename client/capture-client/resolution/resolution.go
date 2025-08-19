package resolution

import (
	"fmt"
	"strconv"
	"strings"
)

type Resolution struct {
	Width  int
	Height int
}

func EmptyResolution() Resolution {
	return Resolution{Width: 0, Height: 0}
}

func Resolution240p() Resolution {
	return Resolution{Width: 426, Height: 240}
}
func Resolution360p() Resolution {
	return Resolution{Width: 640, Height: 360}
}
func Resolution480p() Resolution {
	return Resolution{Width: 854, Height: 480}
}
func Resolution720p() Resolution {
	return Resolution{Width: 1280, Height: 720}
}
func Resolution1080p() Resolution {
	return Resolution{Width: 1920, Height: 1080}
}

// Returns the string representation of this Resolution (e.g. 640x480)
func (r Resolution) String() string {
	return fmt.Sprintf("%dx%d", r.Width, r.Height)
}

func (r Resolution) Format(formatString string) string {
	// replace "w" with width and "h" with height
	result := strings.Replace(formatString, "w", fmt.Sprint(r.Width), -1)
	result = strings.Replace(result, "h", fmt.Sprint(r.Height), -1)
	return result
}

func (r Resolution) AspectRation() float64 {
	if r.Height == 0 {
		return 0
	}
	return float64(r.Width) / float64(r.Height)
}

// IsEmpty checks if the resolution is empty (both width and height are zero).
func (r Resolution) IsEmpty() bool {
	return r.Width == 0 && r.Height == 0
}

// Parse converts a string representation of a resolution (e.g., "1920x1080") into a Resolution struct.
// Supported formats:
// - "1920x1080"
// - "1920:1080"
// - "1080p" (interpreted as 1920x1080)
// - "720p" (interpreted as 1280x720)
func Parse(resolutionStr string) (Resolution, error) {
	var res Resolution
	var err error
	switch {
	case strings.Contains(resolutionStr, "x"):
		res, err = parseDimensions(resolutionStr)
	case strings.Contains(resolutionStr, ":"):
		res, err = parseDimensions(strings.ReplaceAll(resolutionStr, ":", "x"))
	case strings.HasSuffix(resolutionStr, "p"):
		res, err = parsePreset(resolutionStr)
	default:
		err = fmt.Errorf("invalid resolution format: %s", resolutionStr)
	}
	if err != nil {
		return Resolution{}, err
	}
	return res, nil
}
func parseDimensions(dimStr string) (Resolution, error) {
	parts := strings.Split(dimStr, "x")
	if len(parts) != 2 {
		return Resolution{}, fmt.Errorf("invalid dimensions: %s", dimStr)
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return Resolution{}, fmt.Errorf("invalid width: %s", parts[0])
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return Resolution{}, fmt.Errorf("invalid height: %s", parts[1])
	}

	return Resolution{Width: width, Height: height}, nil

}

func parsePreset(preset string) (Resolution, error) {
	switch preset {
	case "1080p":
		return Resolution1080p(), nil
	case "720p":
		return Resolution720p(), nil
	case "480p":
		return Resolution480p(), nil
	case "360p":
		return Resolution360p(), nil
	case "240p":
		return Resolution240p(), nil
	default:
		return Resolution{}, fmt.Errorf("unsupported resolution preset: %s", preset)
	}
}
