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
	Start(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob), failureCallback func(job *UploadJob))

	// Drain processes remaining uploads during shutdown with timeout
	Drain(timeout time.Duration)
}

// uploadQueue implements UploadService for server uploads
type uploadQueue struct {
	serverClient client.CaptureServerClient
	uploadQueue  chan *UploadJob
	retryQueue   chan *UploadJob
	bufferSize   int
	drainTimeout time.Duration
	retryMinutes int
	maxRetries   int
}

// NewUploadQueue creates a new server upload service
func NewUploadQueue(serverClient client.CaptureServerClient, bufferSize int, retryBufferSize int, drainTimeout time.Duration, retryMinutes int, maxRetries int) UploadQueue {
	return &uploadQueue{
		serverClient: serverClient,
		uploadQueue:  make(chan *UploadJob, bufferSize),
		retryQueue:   make(chan *UploadJob, retryBufferSize),
		bufferSize:   bufferSize,
		drainTimeout: drainTimeout,
		retryMinutes: retryMinutes,
		maxRetries:   maxRetries,
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
func (s *uploadQueue) Start(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
	defer wg.Done()

	// Start retry timer goroutine
	wg.Add(1)
	go s.retryWorker(stopChan, wg, successCallback, failureCallback)

	for {
		select {
		case uploadJob := <-s.uploadQueue:
			s.uploadClip(uploadJob, successCallback, failureCallback)
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
			s.uploadClip(uploadJob, successCallback, failureCallback)
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
func (s *uploadQueue) uploadClip(job *UploadJob, successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
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
		if job.RetryCount < s.maxRetries {
			job.RetryCount++
			job.NextRetryTime = time.Now().Add(time.Duration(s.retryMinutes) * time.Minute)
			log.Printf("Scheduling retry for %s in %d minutes (attempt %d/%d)",
				job.FilePath, s.retryMinutes, job.RetryCount, s.maxRetries)

			// Add to retry queue
			select {
			case s.retryQueue <- job:
				// Successfully queued for retry
			default:
				// Retry queue full, give up and call failure callback
				log.Printf("Retry queue full, giving up on %s", job.FilePath)
				if failureCallback != nil {
					failureCallback(job)
				}
			}
		} else {
			// Max retries exceeded, call failure callback
			log.Printf("Max retries exceeded for %s, giving up", job.FilePath)
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

// retryWorker handles retrying failed uploads after the configured delay
func (s *uploadQueue) retryWorker(stopChan <-chan struct{}, wg *sync.WaitGroup, successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
	defer wg.Done()

	ticker := time.NewTicker(1 * time.Minute) // Check every minute for retries
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Check retry queue for jobs ready to retry
			s.processRetryQueue(successCallback, failureCallback)
		}
	}
}

// processRetryQueue processes jobs in the retry queue that are ready to retry
func (s *uploadQueue) processRetryQueue(successCallback func(job *UploadJob), failureCallback func(job *UploadJob)) {
	now := time.Now()

	// Process all jobs in retry queue
	for {
		select {
		case retryJob := <-s.retryQueue:
			if now.After(retryJob.NextRetryTime) || now.Equal(retryJob.NextRetryTime) {
				// Time to retry
				log.Printf("Retrying upload for %s (attempt %d/%d)", retryJob.FilePath, retryJob.RetryCount, s.maxRetries)
				s.uploadClip(retryJob, successCallback, failureCallback)
			} else {
				// Not time yet, put back in retry queue
				select {
				case s.retryQueue <- retryJob:
					// Successfully requeued
				default:
					// Retry queue full, give up
					log.Printf("Retry queue full while requeuing %s, giving up", retryJob.FilePath)
					if failureCallback != nil {
						failureCallback(retryJob)
					}
				}
			}
		default:
			// No more jobs in retry queue
			return
		}
	}
}
