package motiondetection

import (
	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/config"
)

type MotionDetectionSettings struct {
	MotionMinArea    int // Minimum area of motion to be detected
	MaxFramesToCheck int // Maximum number of frames to check for motion
	WarmUpFrames     int // Number of frames to skip before starting motion detection
}

// MotionDetectionSettingsProvider implements SettingsProvider for MotionDetectionSettings
type MotionDetectionSettingsProvider struct {
	clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]
}

// NewMotionDetectionSettingsProvider creates a new MotionDetectionSettingsProvider
func NewMotionDetectionSettingsProvider(clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]) *MotionDetectionSettingsProvider {
	return &MotionDetectionSettingsProvider{
		clientSettingsProvider: clientSettingsProvider,
	}
}

// GetSettings returns the current motion detection settings mapped from client settings
func (p *MotionDetectionSettingsProvider) GetSettings() MotionDetectionSettings {
	clientSettings := p.clientSettingsProvider.GetSettings()

	return MotionDetectionSettings{
		MotionMinArea:    clientSettings.MotionMinArea,
		MaxFramesToCheck: clientSettings.MotionMaxFrames,
		WarmUpFrames:     clientSettings.MotionWarmUpFrames,
	}
}
