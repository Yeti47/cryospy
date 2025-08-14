package models

import "time"

// ClientSettings represents the client settings fetched from the server
type ClientSettings struct {
	ID                    string `json:"id"`
	StorageLimitMegabytes int    `json:"storage_limit_megabytes"`
	ClipDurationSeconds   int    `json:"clip_duration_seconds"`
	MotionOnly            bool   `json:"motion_only"`
	Grayscale             bool   `json:"grayscale"`
	DownscaleResolution   string `json:"downscale_resolution"`
}

// ChunkDuration returns the chunk duration as a time.Duration
func (cs *ClientSettings) ChunkDuration() time.Duration {
	return time.Duration(cs.ClipDurationSeconds) * time.Second
}

// VideoClip represents a recorded video clip
type VideoClip struct {
	Filename  string
	Timestamp time.Time
	Duration  time.Duration
	HasMotion bool
	FilePath  string
}
