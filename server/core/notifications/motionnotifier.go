package notifications

import (
	"fmt"
	"sync"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

type MotionNotifier interface {
	// NotifyMotionDetected sends a notification when motion is detected.
	NotifyMotionDetected(clientID string, clipTitle string, timestamp time.Time) error
}

type nopMotionNotifier struct{}

var NopMotionNotifier MotionNotifier = &nopMotionNotifier{}

// NotifyMotionDetected does nothing and returns nil.
func (n *nopMotionNotifier) NotifyMotionDetected(clientID string, clipTitle string, timestamp time.Time) error {
	// No operation performed
	return nil
}

type MotionNotificationSettings struct {
	Recipient   string
	MinInterval time.Duration
}

type emailMotionNotifier struct {
	settings          MotionNotificationSettings
	sender            EmailSender
	logger            logging.Logger
	lastNotification  map[string]time.Time
	notificationMutex sync.Mutex
}

func NewEmailMotionNotifier(settings MotionNotificationSettings, sender EmailSender, logger logging.Logger) MotionNotifier {
	return &emailMotionNotifier{
		settings:         settings,
		sender:           sender,
		logger:           logger,
		lastNotification: make(map[string]time.Time),
	}
}

func (n *emailMotionNotifier) NotifyMotionDetected(clientID string, clipTitle string, timestamp time.Time) error {
	n.notificationMutex.Lock()
	defer n.notificationMutex.Unlock()

	if time.Since(n.lastNotification[clientID]) < n.settings.MinInterval {
		n.logger.Info("Skipping motion notification due to rate limiting.", "client", clientID)
		return nil
	}

	subject := "CryoSpy motion detected"
	body := fmt.Sprintf("Motion was detected by client '%s' at %s.\n\nClip: %s\n\nPlease check the dashboard for more details.",
		clientID,
		timestamp.Format("2006-01-02 15:04:05 UTC"),
		clipTitle)

	n.logger.Info("Sending motion detection notification.", "client", clientID, "recipient", n.settings.Recipient)
	err := n.sender.SendEmail(n.settings.Recipient, subject, body)
	if err != nil {
		n.logger.Error("Failed to send motion detection notification.", "error", err, "client", clientID)
		return err
	}

	n.lastNotification[clientID] = time.Now()
	return nil
}
