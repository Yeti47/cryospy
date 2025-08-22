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

// ShouldNotify does nothing and returns false.
func (n *nopAuthNotifier) ShouldNotify(failureCount int) bool {
	return false
}

type AuthNotificationSettings struct {
	Recipient        string
	MinInterval      time.Duration
	FailureThreshold int
}

type emailAuthNotifier struct {
	settings          AuthNotificationSettings
	sender            EmailSender
	logger            logging.Logger
	lastNotification  map[string]time.Time
	notificationMutex sync.Mutex
}

func NewEmailAuthNotifier(settings AuthNotificationSettings, sender EmailSender, logger logging.Logger) AuthNotifier {
	return &emailAuthNotifier{
		settings:         settings,
		sender:           sender,
		logger:           logger,
		lastNotification: make(map[string]time.Time),
	}
}

func (n *emailAuthNotifier) ShouldNotify(failureCount int) bool {
	return failureCount >= n.settings.FailureThreshold
}

func (n *emailAuthNotifier) NotifyRepeatedAuthFailure(clientID string, failureCount int, clientIP string) error {
	n.notificationMutex.Lock()
	defer n.notificationMutex.Unlock()

	if time.Since(n.lastNotification[clientID]) < n.settings.MinInterval {
		n.logger.Info("Skipping authentication failure notification due to rate limiting.", "client", clientID)
		return nil
	}

	subject := "CryoSpy repeated authentication failures detected"
	body := fmt.Sprintf("Repeated authentication failures detected for client '%s'.\n\nFailure count: %d\nClient IP: %s\n\nThis may indicate a brute force attack or misconfigured client. Please investigate and consider updating client credentials or blocking the IP address if necessary.",
		clientID,
		failureCount,
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
