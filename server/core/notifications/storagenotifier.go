package notifications

import (
	"fmt"
	"sync"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

type StorageNotifier interface {
	// NotifyCapacityReached notifies when the storage capacity is reached.
	NotifyCapacityReached(clientID string, usedMegaBytes int64, totalMegaBytes int64) error
	// NotifyCapacityWarning notifies when the storage capacity is nearing its limit.
	NotifyCapacityWarning(clientID string, usedMegaBytes int64, totalMegaBytes int64) error
	// ShouldWarn returns true if the storage usage is above the warning threshold.
	ShouldWarn(usedMegaBytes int64, totalMegaBytes int64) bool
}

type nopStorageNotifier struct{}

var NopStorageNotifier StorageNotifier = &nopStorageNotifier{}

// NotifyCapacityReached does nothing and returns nil.
func (n *nopStorageNotifier) NotifyCapacityReached(clientID string, usedMegaBytes int64, totalMegaBytes int64) error {
	// No operation performed
	return nil
}

// NotifyCapacityWarning does nothing and returns nil.
func (n *nopStorageNotifier) NotifyCapacityWarning(clientID string, usedMegaBytes int64, totalMegaBytes int64) error {
	// No operation performed
	return nil
}

// ShouldWarn does nothing and returns false.
func (n *nopStorageNotifier) ShouldWarn(usedMegaBytes int64, totalMegaBytes int64) bool {
	return false
}

type StorageNotificationSettings struct {
	Recipient        string
	MinInterval      time.Duration
	WarningThreshold float64
}

type emailStorageNotifier struct {
	settings          StorageNotificationSettings
	sender            EmailSender
	logger            logging.Logger
	lastNotification  map[string]time.Time
	notificationMutex sync.Mutex
	lastWarning       map[string]time.Time
	warningMutex      sync.Mutex
}

func NewEmailStorageNotifier(settings StorageNotificationSettings, sender EmailSender, logger logging.Logger) StorageNotifier {
	return &emailStorageNotifier{
		settings:         settings,
		sender:           sender,
		logger:           logger,
		lastNotification: make(map[string]time.Time),
		lastWarning:      make(map[string]time.Time),
	}
}

func (n *emailStorageNotifier) ShouldWarn(usedMegaBytes int64, totalMegaBytes int64) bool {
	if totalMegaBytes == 0 {
		return false
	}
	usagePercent := float64(usedMegaBytes) / float64(totalMegaBytes)
	return usagePercent >= n.settings.WarningThreshold
}

func (n *emailStorageNotifier) NotifyCapacityReached(clientID string, usedMegaBytes int64, totalMegaBytes int64) error {
	n.notificationMutex.Lock()
	defer n.notificationMutex.Unlock()

	if time.Since(n.lastNotification[clientID]) < n.settings.MinInterval {
		n.logger.Info("Skipping storage capacity notification due to rate limiting.", "client", clientID)
		return nil
	}

	subject := "CryoSpy client storage capacity reached"
	body := fmt.Sprintf("Storage capacity for client '%s' has been reached.\n\nUsed: %d MB\nTotal: %d MB\n\nOld video footage will now be overwritten until capacity is freed.",
		clientID,
		usedMegaBytes,
		totalMegaBytes)

	n.logger.Info("Sending storage capacity reached notification.", "client", clientID, "recipient", n.settings.Recipient)
	err := n.sender.SendEmail(n.settings.Recipient, subject, body)
	if err != nil {
		n.logger.Error("Failed to send storage capacity reached notification.", "error", err, "client", clientID)
		return err
	}

	n.lastNotification[clientID] = time.Now()
	return nil
}

func (n *emailStorageNotifier) NotifyCapacityWarning(clientID string, usedMegaBytes int64, totalMegaBytes int64) error {
	n.warningMutex.Lock()
	defer n.warningMutex.Unlock()

	if time.Since(n.lastWarning[clientID]) < n.settings.MinInterval {
		n.logger.Info("Skipping storage capacity warning due to rate limiting.", "client", clientID)
		return nil
	}

	subject := "CryoSpy client storage capacity warning"
	body := fmt.Sprintf("Storage capacity for client '%s' is nearing its limit.\n\nUsed: %d MB\nTotal: %d MB\n\nPlease consider freeing up space to avoid overwriting old footage.",
		clientID,
		usedMegaBytes,
		totalMegaBytes)

	n.logger.Info("Sending storage capacity warning notification.", "client", clientID, "recipient", n.settings.Recipient)
	err := n.sender.SendEmail(n.settings.Recipient, subject, body)
	if err != nil {
		n.logger.Error("Failed to send storage capacity warning notification.", "error", err, "client", clientID)
		return err
	}

	n.lastWarning[clientID] = time.Now()
	return nil
}
