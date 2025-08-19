package recording

import (
	"time"

	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/config"
)

type RecordingSettings struct {
	ClipDuration time.Duration // Duration of each recorded clip
	Codec        string        // Video codec to use (e.g., "MJPG", "H264")
	FrameRate    float64       // Frame rate for video capture
}

// RecordingSettingsProvider implements SettingsProvider for RecordingSettings
type RecordingSettingsProvider struct {
	clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]
}

// NewRecordingSettingsProvider creates a new RecordingSettingsProvider
func NewRecordingSettingsProvider(clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]) *RecordingSettingsProvider {
	return &RecordingSettingsProvider{
		clientSettingsProvider: clientSettingsProvider,
	}
}

// GetSettings returns the current recording settings mapped from client settings
func (p *RecordingSettingsProvider) GetSettings() RecordingSettings {
	clientSettings := p.clientSettingsProvider.GetSettings()

	return RecordingSettings{
		ClipDuration: time.Duration(clientSettings.ClipDurationSeconds) * time.Second,
		Codec:        clientSettings.CaptureCodec,
		FrameRate:    clientSettings.CaptureFrameRate,
	}
}
