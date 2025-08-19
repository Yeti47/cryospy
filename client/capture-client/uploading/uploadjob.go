package uploading

import (
	"time"

	postprocessing "github.com/yeti47/cryospy/client/capture-client/post-processing"
)

// UploadJob represents a video clip ready for upload
type UploadJob struct {
	FilePath           string
	HasMotion          bool
	Duration           time.Duration
	RecordingTimestamp time.Time
	ProcessedClip      *postprocessing.VideoClip
}
