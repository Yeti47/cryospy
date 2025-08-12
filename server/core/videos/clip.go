package videos

import (
	"time"
)

type Clip struct {
	ID                 string
	ClientID           string
	Title              string
	TimeStamp          time.Time
	Duration           time.Duration
	HasMotion          bool
	EncryptedVideo     []byte
	VideoWidth         int
	VideoHeight        int
	VideoMimeType      string
	EncryptedThumbnail []byte
	ThumbnailWidth     int
	ThumbnailHeight    int
	ThumbnailMimeType  string
}

// ClipInfo represents metadata about a clip without the actual video data
type ClipInfo struct {
	ID                string
	ClientID          string
	Title             string
	TimeStamp         time.Time
	Duration          time.Duration
	HasMotion         bool
	VideoWidth        int
	VideoHeight       int
	VideoMimeType     string
	ThumbnailWidth    int
	ThumbnailHeight   int
	ThumbnailMimeType string
}

// ClipQuery represents query parameters for searching clips
type ClipQuery struct {
	ClientID  string // empty string means no filter, otherwise filter by specific client
	StartTime *time.Time
	EndTime   *time.Time
	HasMotion *bool // nil means no filter, true/false means filter by motion
	Limit     *int  // maximum number of records to return (nil means no limit)
	Offset    *int  // number of records to skip (nil means no offset)
}

// DecryptedClip represents a clip with decrypted video and thumbnail data
type DecryptedClip struct {
	ID                string
	ClientID          string
	Title             string
	TimeStamp         time.Time
	Duration          time.Duration
	HasMotion         bool
	Video             []byte // Decrypted video data
	VideoWidth        int
	VideoHeight       int
	VideoMimeType     string
	Thumbnail         []byte // Decrypted thumbnail data (nil if no thumbnail)
	ThumbnailWidth    int
	ThumbnailHeight   int
	ThumbnailMimeType string
}

// Thumbnail represents thumbnail data with its metadata
type Thumbnail struct {
	Data     []byte
	Width    int
	Height   int
	MimeType string
}

// VideoMetadata contains extracted video information
type VideoMetadata struct {
	Width     int
	Height    int
	MimeType  string
	Extension string
}
