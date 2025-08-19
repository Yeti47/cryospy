package uploading

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/client"
	"github.com/yeti47/cryospy/client/capture-client/common"
)

// UploadQueue handles uploading video clips to the server
type UploadQueue interface {
	// Queue adds an upload job to the queue
	Queue(job *UploadJob) bool

	// Start begins processing the upload queue
	Start(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob))

	// Drain processes remaining uploads during shutdown with timeout
	Drain(timeout time.Duration)
}

// uploadQueue implements UploadService for server uploads
type uploadQueue struct {
	serverClient client.CaptureServerClient
	uploadQueue  chan *UploadJob
	bufferSize   int
	drainTimeout time.Duration
}

// NewUploadQueue creates a new server upload service
func NewUploadQueue(serverClient client.CaptureServerClient, bufferSize int, drainTimeout time.Duration) UploadQueue {
	return &uploadQueue{
		serverClient: serverClient,
		uploadQueue:  make(chan *UploadJob, bufferSize),
		bufferSize:   bufferSize,
		drainTimeout: drainTimeout,
	}
}

// QueueUpload adds an upload job to the queue
func (s *uploadQueue) Queue(job *UploadJob) bool {
	select {
	case s.uploadQueue <- job:
		log.Printf("Queued %s for upload (motion: %v, duration: %v)",
			job.FilePath, job.HasMotion, job.Duration)
		return true
	default:
		log.Printf("Upload queue full, dropping %s", job.FilePath)
		return false
	}
}

// Start begins processing the upload queue
func (s *uploadQueue) Start(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob)) {
	defer wg.Done()

	for {
		select {
		case uploadJob := <-s.uploadQueue:
			s.uploadClip(uploadJob, successCallback)
		case <-stopChan:
			// Process remaining uploads with timeout
			s.drainQueueWithCallback(s.drainTimeout, successCallback)
			return
		}
	}
}

// DrainQueue processes remaining uploads during shutdown with timeout
func (s *uploadQueue) Drain(timeout time.Duration) {
	s.drainQueueWithCallback(timeout, nil)
}

// drainQueueWithCallback processes remaining uploads with optional callback
func (s *uploadQueue) drainQueueWithCallback(timeout time.Duration, successCallback func(job *UploadJob)) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case uploadJob := <-s.uploadQueue:
			s.uploadClip(uploadJob, successCallback)
		case <-timer.C:
			log.Println("Upload queue drain timeout, forcing shutdown")
			return
		default:
			// No more uploads in queue
			return
		}
	}
}

// uploadClip uploads a processed video clip to the server
func (s *uploadQueue) uploadClip(job *UploadJob, successCallback func(job *UploadJob)) {
	log.Printf("Uploading %s (motion: %v, duration: %v, recorded: %v)...",
		job.FilePath, job.HasMotion, job.Duration, job.RecordingTimestamp)

	// Read the video file
	videoData, err := os.ReadFile(job.FilePath)
	if err != nil {
		log.Printf("Failed to read processed video %s: %v", job.FilePath, err)
		return
	}

	// Create upload request
	uploadRequest := client.UploadClipRequest{
		VideoData:          videoData,
		MimeType:           common.VideoFormatToMimeType(job.ProcessedClip.Format),
		Duration:           job.Duration,
		HasMotion:          job.HasMotion,
		RecordingTimestamp: job.RecordingTimestamp,
	}

	// Upload to server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = s.serverClient.UploadClip(ctx, uploadRequest)
	if err != nil {
		log.Printf("Failed to upload %s: %v", job.FilePath, err)
		// TODO: Implement retry logic or save to disk for later retry
		return
	}

	log.Printf("Successfully uploaded %s", job.FilePath)

	// Call success callback if set
	if successCallback != nil {
		successCallback(job)
	}
}
