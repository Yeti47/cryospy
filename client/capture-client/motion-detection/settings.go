package motiondetection

import (
	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/config"
)

type MotionDetectionSettings struct {
	MotionMinArea      int     // Minimum area of motion to be detected
	MaxFramesToCheck   int     // Maximum number of frames to check for motion
	WarmUpFrames       int     // Number of frames to skip before starting motion detection
	MotionMinWidth     int     // Minimum width of detected motion
	MotionMinHeight    int     // Minimum height of detected motion
	MotionMinAspect    float64 // Minimum aspect ratio of detected motion
	MotionMaxAspect    float64 // Maximum aspect ratio of detected motion
	MotionMogHistory   int     // MOG2 history parameter
	MotionMogVarThresh float64 // MOG2 var threshold parameter
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
		MotionMinArea:      clientSettings.MotionMinArea,
		MaxFramesToCheck:   clientSettings.MotionMaxFrames,
		WarmUpFrames:       clientSettings.MotionWarmUpFrames,
		MotionMinWidth:     clientSettings.MotionMinWidth,
		MotionMinHeight:    clientSettings.MotionMinHeight,
		MotionMinAspect:    clientSettings.MotionMinAspect,
		MotionMaxAspect:    clientSettings.MotionMaxAspect,
		MotionMogHistory:   clientSettings.MotionMogHistory,
		MotionMogVarThresh: clientSettings.MotionMogVarThresh,
	}
}
