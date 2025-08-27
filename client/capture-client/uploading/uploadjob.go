package uploading

import (
	"time"
)

// UploadJob represents a video clip ready for upload
type UploadJob struct {
	FilePath           string
	HasMotion          bool
	Duration           time.Duration
	RecordingTimestamp time.Time
	Format             string // Video format (e.g., "mp4", "avi") for MIME type determination
	RetryCount         int    // Number of retry attempts made
}
