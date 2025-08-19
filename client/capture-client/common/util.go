package common

import (
	"strings"
)

func CodecToFileExtension(codec string) string {
	codec = strings.ToUpper(codec)
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

// GetOutputMimeType returns the MIME type for the configured video output format
func VideoFormatToMimeType(format string) string {
	format = strings.ToLower(format)
	format = strings.TrimPrefix(format, ".") // Remove leading dot if present
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
