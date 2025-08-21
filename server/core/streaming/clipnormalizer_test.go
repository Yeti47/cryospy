package streaming

import (
	"os"
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
)

func TestFFmpegClipNormalizer_NormalizeClip_Validation(t *testing.T) {
	normalizer := NewFFmpegClipNormalizer(logging.NopLogger, "")

	tests := []struct {
		name    string
		clip    *videos.DecryptedClip
		wantErr bool
	}{
		{
			name:    "nil clip",
			clip:    nil,
			wantErr: true,
		},
		{
			name: "empty video data",
			clip: &videos.DecryptedClip{
				ID:    "test-id",
				Video: []byte{},
			},
			wantErr: true,
		},
		{
			name: "empty ID",
			clip: &videos.DecryptedClip{
				ID:    "",
				Video: []byte("test video data"),
			},
			wantErr: true,
		},
		{
			name: "valid clip structure but fake video data",
			clip: &videos.DecryptedClip{
				ID:    "test-id",
				Video: []byte("test video data"), // This will fail transcoding but pass validation
			},
			wantErr: true, // FFmpeg will reject fake video data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizer.NormalizeClip(tt.clip)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeClip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultNormalizationSettings(t *testing.T) {
	settings := DefaultNormalizationSettings()

	// Verify sensible defaults
	if settings.Width != 854 {
		t.Errorf("Expected width 854, got %d", settings.Width)
	}
	if settings.Height != 480 {
		t.Errorf("Expected height 480, got %d", settings.Height)
	}
	if settings.VideoBitrate != "1000k" {
		t.Errorf("Expected video bitrate 1000k, got %s", settings.VideoBitrate)
	}
	if settings.VideoCodec != "libx264" {
		t.Errorf("Expected video codec libx264, got %s", settings.VideoCodec)
	}
	if settings.FrameRate != 25 {
		t.Errorf("Expected frame rate 25, got %d", settings.FrameRate)
	}
}

func TestNewFFmpegClipNormalizer(t *testing.T) {
	tempDir := "/tmp/test"
	normalizer := NewFFmpegClipNormalizer(nil, tempDir)

	if normalizer.logger == nil {
		t.Error("Expected logger to be set (NopLogger)")
	}
	if normalizer.tempDir != tempDir {
		t.Errorf("Expected tempDir %s, got %s", tempDir, normalizer.tempDir)
	}

	defaultSettings := DefaultNormalizationSettings()
	if normalizer.settings != defaultSettings {
		t.Error("Expected default settings to be set")
	}
}

func TestNewFFmpegClipNormalizerWithSettings(t *testing.T) {
	tempDir := "/tmp/test"
	customSettings := NormalizationSettings{
		Width:        1280,
		Height:       720,
		VideoBitrate: "2000k",
		VideoCodec:   "libx265",
		FrameRate:    30,
	}

	normalizer := NewFFmpegClipNormalizerWithSettings(nil, tempDir, customSettings)

	if normalizer.settings != customSettings {
		t.Error("Expected custom settings to be set")
	}
}

// Mock test for NormalizeClip - this would need a real video file to test properly
func TestFFmpegClipNormalizer_NormalizeClip_MockValidation(t *testing.T) {
	normalizer := NewFFmpegClipNormalizer(logging.NopLogger, "")

	// Test with invalid clip
	_, err := normalizer.NormalizeClip(nil)
	if err == nil {
		t.Error("Expected error for nil clip")
	}

	// Test with empty video data
	clip := &videos.DecryptedClip{
		ID:            "test-clip",
		ClientID:      "test-client",
		Title:         "Test Clip",
		TimeStamp:     time.Now(),
		Duration:      time.Minute,
		HasMotion:     true,
		Video:         []byte{}, // Empty video data
		VideoWidth:    1920,
		VideoHeight:   1080,
		VideoMimeType: "video/mp4",
	}

	_, err = normalizer.NormalizeClip(clip)
	if err == nil {
		t.Error("Expected error for empty video data")
	}
}

// Integration test - only run if ffmpeg is available and test video exists
func TestFFmpegClipNormalizer_NormalizeClip_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if ffmpeg is available
	if _, err := os.Stat("/usr/bin/ffmpeg"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/local/bin/ffmpeg"); os.IsNotExist(err) {
			t.Skip("ffmpeg not found, skipping integration test")
		}
	}

	normalizer := NewFFmpegClipNormalizer(logging.NopLogger, os.TempDir())

	// This would need a real video file to test with
	// For now, we'll just test the structure
	clip := &videos.DecryptedClip{
		ID:            "test-clip",
		ClientID:      "test-client",
		Title:         "Test Clip",
		TimeStamp:     time.Now(),
		Duration:      time.Minute,
		HasMotion:     true,
		Video:         []byte("fake video data"), // This would need to be real video data
		VideoWidth:    1920,
		VideoHeight:   1080,
		VideoMimeType: "video/mp4",
	}

	// This will fail because we don't have real video data, but it tests the flow
	_, err := normalizer.NormalizeClip(clip)
	// We expect this to fail with fake data, but the structure should be correct
	if err == nil {
		t.Error("Expected error with fake video data")
	}
}
