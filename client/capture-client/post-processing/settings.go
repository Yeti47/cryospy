package postprocessing

import (
	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/config"
	"github.com/yeti47/cryospy/client/capture-client/resolution"
)

type PostProcessingSettings struct {
	OutputFormat        string                // Output container format (e.g., "mp4", "avi")
	OutputCodec         string                // Video codec to use (e.g., "H264")
	VideoBitRate        string                // Bitrate for video compression (e.g., "1000k")
	Grayscale           bool                  // Whether to convert video to grayscale
	DownscaleResolution resolution.Resolution // Resolution to downscale video to (e.g., "1280x720")
}

// PostProcessingSettingsProvider implements SettingsProvider for PostProcessingSettings
type PostProcessingSettingsProvider struct {
	clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]
}

// NewPostProcessingSettingsProvider creates a new PostProcessingSettingsProvider
func NewPostProcessingSettingsProvider(clientSettingsProvider config.SettingsProvider[client.ClientSettingsResponse]) *PostProcessingSettingsProvider {
	return &PostProcessingSettingsProvider{
		clientSettingsProvider: clientSettingsProvider,
	}
}

// GetSettings returns the current post-processing settings mapped from client settings
func (p *PostProcessingSettingsProvider) GetSettings() PostProcessingSettings {
	clientSettings := p.clientSettingsProvider.GetSettings()

	// Parse the downscale resolution string, fallback to empty resolution if parsing fails
	downscaleRes := resolution.EmptyResolution()
	if parsedRes, err := resolution.Parse(clientSettings.DownscaleResolution); err == nil {
		downscaleRes = parsedRes
	}

	return PostProcessingSettings{
		OutputFormat:        clientSettings.OutputFormat,
		OutputCodec:         clientSettings.OutputCodec,
		VideoBitRate:        clientSettings.VideoBitRate,
		Grayscale:           clientSettings.Grayscale,
		DownscaleResolution: downscaleRes,
	}
}
