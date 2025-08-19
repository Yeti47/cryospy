package client

import "time"

// ClientSettingsResponse represents the client settings response from the server
type ClientSettingsResponse struct {
	ID                    string  `json:"id"`
	StorageLimitMegabytes int     `json:"storage_limit_megabytes"`
	ClipDurationSeconds   int     `json:"clip_duration_seconds"`
	MotionOnly            bool    `json:"motion_only"`
	Grayscale             bool    `json:"grayscale"`
	DownscaleResolution   string  `json:"downscale_resolution"`
	OutputFormat          string  `json:"output_format"`
	OutputCodec           string  `json:"output_codec"`
	VideoBitRate          string  `json:"video_bitrate"`
	MotionMinArea         int     `json:"motion_min_area"`
	MotionMaxFrames       int     `json:"motion_max_frames"`
	MotionWarmUpFrames    int     `json:"motion_warm_up_frames"`
	CaptureCodec          string  `json:"capture_codec"`
	CaptureFrameRate      float64 `json:"capture_frame_rate"`
}

type UploadClipRequest struct {
	VideoData          []byte
	MimeType           string
	Duration           time.Duration
	HasMotion          bool
	RecordingTimestamp time.Time
}
