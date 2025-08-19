package filemanagement

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

// FileTracker manages file cleanup operations
type FileTracker interface {
	// DeleteFile removes a file from disk
	DeleteFile(filePath string)

	// EnsureTempDirectory creates the temporary directory if it doesn't exist
	EnsureTempDirectory() error

	// CleanupTempDirectory removes all files in the temporary directory
	CleanupTempDirectory()
}

// LocalFileTracker implements FileTracker for local filesystem
type LocalFileTracker struct {
	tempDir string
	mu      sync.Mutex
}

// NewLocalFileTracker creates a new local file tracker
func NewLocalFileTracker(tempDir string) *LocalFileTracker {
	return &LocalFileTracker{
		tempDir: tempDir,
	}
}

// DeleteFile removes a file from disk
func (t *LocalFileTracker) DeleteFile(filePath string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove file %s: %v", filePath, err)
	} else {
		log.Printf("Deleted file: %s", filePath)
	}
}

// EnsureTempDirectory creates the temporary directory if it doesn't exist
func (t *LocalFileTracker) EnsureTempDirectory() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := os.MkdirAll(t.tempDir, 0755); err != nil {
		return err
	}
	log.Printf("Temporary directory ready: %s", t.tempDir)
	return nil
}

// CleanupTempDirectory removes all files in the temporary directory
func (t *LocalFileTracker) CleanupTempDirectory() {
	t.mu.Lock()
	defer t.mu.Unlock()

	entries, err := os.ReadDir(t.tempDir)
	if err != nil {
		log.Printf("Failed to read temp directory %s: %v", t.tempDir, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(t.tempDir, entry.Name())
			log.Printf("Cleaning up temp file: %s", filePath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("Failed to remove temp file %s: %v", filePath, err)
			}
		}
	}
}
