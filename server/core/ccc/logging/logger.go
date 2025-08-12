package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type LogLevel string

const (
	// LogLevelDebug is used for debug messages
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo is used for informational messages
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn is used for warning messages
	LogLevelWarn LogLevel = "warn"
	// LogLevelError is used for error messages
	LogLevelError LogLevel = "error"
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
func newDailyRotatingWriter(logDir, filename string) *dailyRotatingWriter {
	return &dailyRotatingWriter{
		logDir:   logDir,
		filename: filename,
	}
}

// Write implements the io.Writer interface
func (w *dailyRotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// for logging, we want to use local time
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
	filepath := filepath.Join(w.logDir, filename)

	// Open new file
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

// CreateLogger creates a logger that writes to daily rotating log files
func CreateLogger(logLevel LogLevel, logDir string, fileName string) Logger {

	var level slog.Level
	switch logLevel {
	case LogLevelDebug:
		level = slog.LevelDebug
	case LogLevelInfo:
		level = slog.LevelInfo
	case LogLevelWarn:
		level = slog.LevelWarn
	case LogLevelError:
		level = slog.LevelError
	default:
		// Default to Info if unknown level
		level = slog.LevelInfo
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fallback to console logging if we can't create the log directory
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}))
	}

	// Create daily rotating writer
	rotatingWriter := newDailyRotatingWriter(logDir, fileName)

	return slog.New(slog.NewJSONHandler(rotatingWriter, &slog.HandlerOptions{
		Level: level,
	}))
}

// nopLogger is a no-operation logger that implements the Logger interface.
type nopLogger struct{}

// NopLogger is a singleton Logger that performs no operations.
// Use this when no logging is desired or when a logger is required but no output is needed.
var NopLogger Logger = &nopLogger{}

// Info implements the Logger interface for nopLogger.
func (l *nopLogger) Info(msg string, args ...any) {}

// Warn implements the Logger interface for nopLogger.
func (l *nopLogger) Warn(msg string, args ...any) {}

// Error implements the Logger interface for nopLogger.
func (l *nopLogger) Error(msg string, args ...any) {}

// Debug implements the Logger interface for nopLogger.
func (l *nopLogger) Debug(msg string, args ...any) {}
