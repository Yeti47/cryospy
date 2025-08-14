package utils

import (
	"bytes"
	"fmt"
)

// VideoMagicBytes contains common video file magic byte signatures
var VideoMagicBytes = map[string][]byte{
	"mp4":  {0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}, // MP4 container
	"mp4a": {0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, // Alternative MP4
	"avi":  {0x52, 0x49, 0x46, 0x46},                         // RIFF header (AVI)
	"mov":  {0x00, 0x00, 0x00, 0x14, 0x66, 0x74, 0x79, 0x70}, // QuickTime MOV
	"wmv":  {0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11}, // Windows Media Video
	"flv":  {0x46, 0x4C, 0x56, 0x01},                         // Flash Video
	"mkv":  {0x1A, 0x45, 0xDF, 0xA3},                         // Matroska Video
	"webm": {0x1A, 0x45, 0xDF, 0xA3},                         // WebM (same as Matroska)
}

// MP4Brands contains common MP4 brand signatures that can appear after ftyp
var MP4Brands = [][]byte{
	{0x69, 0x73, 0x6F, 0x6D}, // isom
	{0x6D, 0x70, 0x34, 0x31}, // mp41
	{0x6D, 0x70, 0x34, 0x32}, // mp42
	{0x61, 0x76, 0x63, 0x31}, // avc1
	{0x64, 0x61, 0x73, 0x68}, // dash
	{0x6D, 0x70, 0x34, 0x76}, // mp4v
	{0x4D, 0x34, 0x41, 0x20}, // M4A
}

// IsVideoFile checks if the provided data appears to be a video file
// by examining the magic bytes at the beginning of the file
func IsVideoFile(data []byte) (bool, string, error) {
	if len(data) < 12 {
		return false, "", fmt.Errorf("data too short to determine file type")
	}

	// Check for standard video magic bytes
	for format, magic := range VideoMagicBytes {
		if len(data) >= len(magic) && bytes.Equal(data[:len(magic)], magic) {
			// For MP4-based formats, also check the brand
			if format == "mp4" || format == "mp4a" || format == "mov" {
				if isValidMP4Brand(data) {
					return true, format, nil
				}
			} else {
				return true, format, nil
			}
		}
	}

	// Special handling for MP4 files with different box sizes
	if len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")) {
		if isValidMP4Brand(data) {
			return true, "mp4", nil
		}
	}

	// Check for AVI files (RIFF header followed by AVI)
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("AVI ")) {
		return true, "avi", nil
	}

	return false, "", nil
}

// isValidMP4Brand checks if the MP4 file has a valid brand identifier
func isValidMP4Brand(data []byte) bool {
	if len(data) < 12 {
		return false
	}

	// Brand starts at offset 8 in MP4 files
	brandOffset := 8
	if len(data) < brandOffset+4 {
		return false
	}

	brand := data[brandOffset : brandOffset+4]
	for _, validBrand := range MP4Brands {
		if bytes.Equal(brand, validBrand) {
			return true
		}
	}

	return false
}
