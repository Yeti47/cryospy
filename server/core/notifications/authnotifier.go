package notifications

import (
	"fmt"
	"sync"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

type AuthNotifier interface {
	// NotifyRepeatedAuthFailure notifies when repeated authentication failures are detected.
	NotifyRepeatedAuthFailure(clientID string, failureCount int, clientIP string) error
	// RecordAuthFailure records an authentication failure and returns the current failure count within the time window.
	RecordAuthFailure(clientID string, clientIP string, timestamp time.Time) int
	// ShouldNotify returns true if the failure count exceeds the threshold within the time window.
	ShouldNotify(failureCount int) bool
}

type nopAuthNotifier struct{}

var NopAuthNotifier AuthNotifier = &nopAuthNotifier{}

// NotifyRepeatedAuthFailure does nothing and returns nil.
func (n *nopAuthNotifier) NotifyRepeatedAuthFailure(clientID string, failureCount int, clientIP string) error {
	// No operation performed
	return nil
}

// RecordAuthFailure does nothing and returns 0.
func (n *nopAuthNotifier) RecordAuthFailure(clientID string, clientIP string, timestamp time.Time) int {
	return 0
}

// ShouldNotify does nothing and returns false.
func (n *nopAuthNotifier) ShouldNotify(failureCount int) bool {
	return false
}

type AuthFailureRecord struct {
	ClientID  string
	ClientIP  string
	Timestamp time.Time
}

type AuthNotificationSettings struct {
	Recipient        string
	MinInterval      time.Duration
	FailureThreshold int
	TimeWindow       time.Duration
}

type emailAuthNotifier struct {
	settings          AuthNotificationSettings
	sender            EmailSender
	logger            logging.Logger
	lastNotification  map[string]time.Time
	notificationMutex sync.Mutex
	authFailures      []AuthFailureRecord
	failuresMutex     sync.Mutex
}

func NewEmailAuthNotifier(settings AuthNotificationSettings, sender EmailSender, logger logging.Logger) AuthNotifier {
	return &emailAuthNotifier{
		settings:         settings,
		sender:           sender,
		logger:           logger,
		lastNotification: make(map[string]time.Time),
		authFailures:     make([]AuthFailureRecord, 0),
	}
}

func (n *emailAuthNotifier) ShouldNotify(failureCount int) bool {
	return failureCount >= n.settings.FailureThreshold
}

func (n *emailAuthNotifier) RecordAuthFailure(clientID string, clientIP string, timestamp time.Time) int {
	n.failuresMutex.Lock()
	defer n.failuresMutex.Unlock()

	// Add the new failure record
	record := AuthFailureRecord{
		ClientID:  clientID,
		ClientIP:  clientIP,
		Timestamp: timestamp,
	}
	n.authFailures = append(n.authFailures, record)

	// Clean up old records outside the time window
	cutoffTime := timestamp.Add(-n.settings.TimeWindow)
	validFailures := make([]AuthFailureRecord, 0)
	for _, failure := range n.authFailures {
		if failure.Timestamp.After(cutoffTime) {
			validFailures = append(validFailures, failure)
		}
	}
	n.authFailures = validFailures

	// Count failures for the specific client within the time window
	count := 0
	for _, failure := range n.authFailures {
		if failure.ClientID == clientID && failure.Timestamp.After(cutoffTime) {
			count++
		}
	}

	return count
}

func (n *emailAuthNotifier) NotifyRepeatedAuthFailure(clientID string, failureCount int, clientIP string) error {
	n.notificationMutex.Lock()
	defer n.notificationMutex.Unlock()

	if time.Since(n.lastNotification[clientID]) < n.settings.MinInterval {
		n.logger.Info("Skipping authentication failure notification due to rate limiting.", "client", clientID)
		return nil
	}

	subject := "CryoSpy repeated authentication failures detected"
	body := fmt.Sprintf("Repeated authentication failures detected for client '%s'.\n\nFailure count: %d\nTime window: %v\nClient IP: %s\n\nThis may indicate a brute force attack or misconfigured client. Please investigate and consider updating client credentials or blocking the IP address if necessary.",
		clientID,
		failureCount,
		n.settings.TimeWindow,
		clientIP)

	n.logger.Info("Sending authentication failure notification.", "client", clientID, "recipient", n.settings.Recipient, "failureCount", failureCount)
	err := n.sender.SendEmail(n.settings.Recipient, subject, body)
	if err != nil {
		n.logger.Error("Failed to send authentication failure notification.", "error", err, "client", clientID)
		return err
	}

	n.lastNotification[clientID] = time.Now()
	return nil
}
