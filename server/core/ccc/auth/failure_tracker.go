package auth

import (
	"sync"
	"time"
)

// FailureRecord represents a single authentication failure
type FailureRecord struct {
	ClientID  string
	ClientIP  string
	Timestamp time.Time
}

// FailureTracker tracks authentication failures for clients
type FailureTracker interface {
	// RecordFailure records an authentication failure and returns the current failure count within the time window
	RecordFailure(clientID string, clientIP string, timestamp time.Time) int
	// ShouldAutoDisable returns true if the failure count exceeds the auto-disable threshold
	ShouldAutoDisable(failureCount int) bool
}

// AutoDisableSettings holds configuration for automatic client disabling
type AutoDisableSettings struct {
	Threshold  int           // Number of failures that trigger auto-disable (0 to disable)
	TimeWindow time.Duration // Time window for counting failures
}

// nopFailureTracker is a no-operation implementation
type nopFailureTracker struct{}

var NopFailureTracker FailureTracker = &nopFailureTracker{}

func (n *nopFailureTracker) RecordFailure(clientID string, clientIP string, timestamp time.Time) int {
	return 0
}

func (n *nopFailureTracker) ShouldAutoDisable(failureCount int) bool {
	return false
}

// memoryFailureTracker implements FailureTracker using in-memory storage
type memoryFailureTracker struct {
	settings      AutoDisableSettings
	failures      []FailureRecord
	failuresMutex sync.Mutex
}

// NewMemoryFailureTracker creates a new in-memory failure tracker
func NewMemoryFailureTracker(settings AutoDisableSettings) FailureTracker {
	return &memoryFailureTracker{
		settings: settings,
		failures: make([]FailureRecord, 0),
	}
}

func (t *memoryFailureTracker) ShouldAutoDisable(failureCount int) bool {
	return t.settings.Threshold > 0 && failureCount >= t.settings.Threshold
}

func (t *memoryFailureTracker) RecordFailure(clientID string, clientIP string, timestamp time.Time) int {
	t.failuresMutex.Lock()
	defer t.failuresMutex.Unlock()

	// Add the new failure record
	record := FailureRecord{
		ClientID:  clientID,
		ClientIP:  clientIP,
		Timestamp: timestamp,
	}
	t.failures = append(t.failures, record)

	// Clean up old records outside the time window
	cutoffTime := timestamp.Add(-t.settings.TimeWindow)
	validFailures := make([]FailureRecord, 0)
	for _, failure := range t.failures {
		if failure.Timestamp.After(cutoffTime) || failure.Timestamp.Equal(cutoffTime) {
			validFailures = append(validFailures, failure)
		}
	}
	t.failures = validFailures

	// Count failures for the specific client within the time window
	count := 0
	for _, failure := range t.failures {
		if failure.ClientID == clientID && (failure.Timestamp.After(cutoffTime) || failure.Timestamp.Equal(cutoffTime)) {
			count++
		}
	}

	return count
}
