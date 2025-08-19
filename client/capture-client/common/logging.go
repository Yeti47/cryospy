package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// dailyRotatingWriter is a writer that creates a new log file each day
type dailyRotatingWriter struct {
	logDir      string
	filename    string
	currentFile *os.File
	currentDate string
	mu          sync.Mutex
}

// newDailyRotatingWriter creates a new daily rotating writer
func NewDailyRotatingWriter(logDir, filename string) *dailyRotatingWriter {
	return &dailyRotatingWriter{
		logDir:   logDir,
		filename: filename,
	}
}

// Write implements the io.Writer interface
func (w *dailyRotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Use local time for logging
	currentDate := time.Now().Format("2006-01-02")

	// Check if we need to rotate (new day or no file open)
	if w.currentFile == nil || w.currentDate != currentDate {
		if err := w.rotate(currentDate); err != nil {
			return 0, err
		}
	}

	return w.currentFile.Write(p)
}

// rotate closes the current file and opens a new one for the given date
func (w *dailyRotatingWriter) rotate(date string) error {
	// Close current file if open
	if w.currentFile != nil {
		w.currentFile.Close()
	}

	// Create new filename with date
	filename := fmt.Sprintf("%s-%s.log", w.filename, date)
	filePath := filepath.Join(w.logDir, filename)

	// Open new file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.currentFile = file
	w.currentDate = date
	return nil
}

// Close closes the current file
func (w *dailyRotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}
