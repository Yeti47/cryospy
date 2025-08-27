package uploading

import (
	"context"
	"log"
	"math/rand"
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
	Start(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob), failureCallback func(job *UploadJob))

	// Drain processes remaining uploads during shutdown with timeout
	Drain(timeout time.Duration)
}

// uploadQueue implements UploadService for server uploads
type uploadQueue struct {
	serverClient    client.CaptureServerClient
	uploadQueue     chan *UploadJob
	bufferSize      int
	drainTimeout    time.Duration
	retryMinutes    int
	maxRetries      int
	retryBufferSize int
	activeRetries   int
	activeRetriesMu sync.Mutex
}

// NewUploadQueue creates a new server upload service
func NewUploadQueue(serverClient client.CaptureServerClient, bufferSize int, retryBufferSize int, drainTimeout time.Duration, retryMinutes int, maxRetries int) UploadQueue {
	return &uploadQueue{
		serverClient:    serverClient,
		uploadQueue:     make(chan *UploadJob, bufferSize),
		bufferSize:      bufferSize,
		drainTimeout:    drainTimeout,
		retryMinutes:    retryMinutes,
		maxRetries:      maxRetries,
		retryBufferSize: retryBufferSize,
	}
}

// Queue adds an upload job to the queue
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
func (s *uploadQueue) Start(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
	defer wg.Done()

	for {
		select {
		case uploadJob := <-s.uploadQueue:
			s.uploadClip(uploadJob, stopChan, successCallback, failureCallback)
		case <-stopChan:
			// Process remaining uploads with timeout
			s.drainQueueWithCallback(s.drainTimeout, successCallback, failureCallback)
			return
		}
	}
}

// DrainQueue processes remaining uploads during shutdown with timeout
func (s *uploadQueue) Drain(timeout time.Duration) {
	s.drainQueueWithCallback(timeout, nil, nil)
}

// drainQueueWithCallback processes remaining uploads with optional callback
func (s *uploadQueue) drainQueueWithCallback(timeout time.Duration, successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
	// Calculate actual timeout based on queue length - each upload could take the full timeout
	queueLength := len(s.uploadQueue)
	actualTimeout := max(timeout*time.Duration(queueLength), timeout)

	log.Printf("Draining upload queue with %d items, timeout: %v", queueLength, actualTimeout)

	timer := time.NewTimer(actualTimeout)
	defer timer.Stop()

	for {
		select {
		case uploadJob := <-s.uploadQueue:
			s.uploadClip(uploadJob, nil, successCallback, failureCallback)
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
func (s *uploadQueue) uploadClip(job *UploadJob, stopChan <-chan struct{}, successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
	log.Printf("Uploading %s (motion: %v, duration: %v, recorded: %v, attempt: %d)...",
		job.FilePath, job.HasMotion, job.Duration, job.RecordingTimestamp, job.RetryCount+1)

	// Read the video file
	videoData, err := os.ReadFile(job.FilePath)
	if err != nil {
		log.Printf("Failed to read processed video %s: %v", job.FilePath, err)
		// File read error is permanent, call failure callback immediately
		if failureCallback != nil {
			failureCallback(job)
		}
		return
	}

	// Create upload request
	uploadRequest := client.UploadClipRequest{
		VideoData:          videoData,
		MimeType:           common.VideoFormatToMimeType(job.Format),
		Duration:           job.Duration,
		HasMotion:          job.HasMotion,
		RecordingTimestamp: job.RecordingTimestamp,
	}

	// Upload to server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = s.serverClient.UploadClip(ctx, uploadRequest)
	if err != nil {
		log.Printf("Failed to upload %s (attempt %d): %v", job.FilePath, job.RetryCount+1, err)

		// Check if we should retry
		if job.RetryCount < s.maxRetries && client.IsRecoverableUploadError(err) {
			s.activeRetriesMu.Lock()
			if s.activeRetries < s.retryBufferSize {
				s.activeRetries++
				s.activeRetriesMu.Unlock()
				job.RetryCount++
				log.Printf("Scheduling retry for %s in %d minutes (attempt %d/%d)",
					job.FilePath, s.retryMinutes, job.RetryCount, s.maxRetries)

				go func(job *UploadJob) {
					defer func() {
						s.activeRetriesMu.Lock()
						s.activeRetries--
						s.activeRetriesMu.Unlock()
					}()
					// add jitter to avoid retry storms
					baseDelay := time.Duration(s.retryMinutes) * time.Minute
					jitter := time.Duration(rand.Intn(16)) * time.Second
					retryDelay := baseDelay + jitter

					select {
					case <-stopChan:
						log.Printf("Shutdown requested, dropping retry for %s", job.FilePath)
						if failureCallback != nil {
							failureCallback(job)
						}
						return
					case <-time.After(retryDelay):
						// Retry after sleep
					}
					// Only retry if not shutting down
					select {
					case <-stopChan:
						log.Printf("Shutdown requested, dropping retry for %s", job.FilePath)
						if failureCallback != nil {
							failureCallback(job)
						}
						return
					default:
						s.uploadClip(job, stopChan, successCallback, failureCallback)
					}
				}(job)
			} else {
				s.activeRetriesMu.Unlock()
				log.Printf("Retry buffer full, dropping retry for %s", job.FilePath)
				if failureCallback != nil {
					failureCallback(job)
				}
			}
		} else {

			// check again why not retrying
			if client.IsRecoverableUploadError(err) {
				// Max retries exceeded
				log.Printf("Max retries exceeded for %s, giving up", job.FilePath)
			} else {
				// Non-recoverable error, no point in retrying
				log.Printf("Non-recoverable error for %s, not retrying", job.FilePath)
			}

			if failureCallback != nil {
				failureCallback(job)
			}
		}
		return
	}

	log.Printf("Successfully uploaded %s", job.FilePath)

	// Call success callback if set
	if successCallback != nil {
		successCallback(job)
	}
}
