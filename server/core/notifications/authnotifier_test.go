package notifications

import (
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

func TestEmailAuthNotifier_RecordAuthFailure(t *testing.T) {
	settings := AuthNotificationSettings{
		Recipient:        "admin@example.com",
		MinInterval:      5 * time.Minute,
		FailureThreshold: 3,
		TimeWindow:       time.Hour,
	}

	mockSender := &mockEmailSender{}
	logger := logging.NopLogger
	notifier := NewEmailAuthNotifier(settings, mockSender, logger)

	clientID := "test-client"
	clientIP := "192.168.1.100"
	now := time.Now()

	// Test recording multiple failures
	count1 := notifier.RecordAuthFailure(clientID, clientIP, now)
	if count1 != 1 {
		t.Errorf("Expected failure count 1, got %d", count1)
	}

	count2 := notifier.RecordAuthFailure(clientID, clientIP, now.Add(1*time.Minute))
	if count2 != 2 {
		t.Errorf("Expected failure count 2, got %d", count2)
	}

	count3 := notifier.RecordAuthFailure(clientID, clientIP, now.Add(2*time.Minute))
	if count3 != 3 {
		t.Errorf("Expected failure count 3, got %d", count3)
	}
}

func TestEmailAuthNotifier_ShouldNotify(t *testing.T) {
	settings := AuthNotificationSettings{
		Recipient:        "admin@example.com",
		MinInterval:      5 * time.Minute,
		FailureThreshold: 3,
		TimeWindow:       time.Hour,
	}

	mockSender := &mockEmailSender{}
	logger := logging.NopLogger
	notifier := NewEmailAuthNotifier(settings, mockSender, logger)

	// Should not notify below threshold
	if notifier.ShouldNotify(2) {
		t.Error("Should not notify with failure count below threshold")
	}

	// Should notify at threshold
	if !notifier.ShouldNotify(3) {
		t.Error("Should notify with failure count at threshold")
	}

	// Should notify above threshold
	if !notifier.ShouldNotify(5) {
		t.Error("Should notify with failure count above threshold")
	}
}

func TestEmailAuthNotifier_NotifyRepeatedAuthFailure(t *testing.T) {
	settings := AuthNotificationSettings{
		Recipient:        "admin@example.com",
		MinInterval:      5 * time.Minute,
		FailureThreshold: 3,
		TimeWindow:       time.Hour,
	}

	mockSender := &mockEmailSender{}
	logger := logging.NopLogger
	notifier := NewEmailAuthNotifier(settings, mockSender, logger)

	clientID := "test-client"
	clientIP := "192.168.1.100"
	failureCount := 5

	// First notification should be sent
	err := notifier.NotifyRepeatedAuthFailure(clientID, failureCount, clientIP)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(mockSender.sentEmails) != 1 {
		t.Errorf("Expected 1 email to be sent, got %d", len(mockSender.sentEmails))
	}

	// Verify email content
	email := mockSender.sentEmails[0]
	if email.to != settings.Recipient {
		t.Errorf("Expected recipient %s, got %s", settings.Recipient, email.to)
	}

	if email.subject != "CryoSpy repeated authentication failures detected" {
		t.Errorf("Unexpected email subject: %s", email.subject)
	}

	// Second notification within min interval should be skipped
	err = notifier.NotifyRepeatedAuthFailure(clientID, failureCount, clientIP)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(mockSender.sentEmails) != 1 {
		t.Errorf("Expected still 1 email (rate limited), got %d", len(mockSender.sentEmails))
	}
}

func TestNopAuthNotifier(t *testing.T) {
	notifier := NopAuthNotifier

	// All methods should return zero/false/nil
	count := notifier.RecordAuthFailure("client", "ip", time.Now())
	if count != 0 {
		t.Errorf("Expected 0 failure count, got %d", count)
	}

	if notifier.ShouldNotify(10) {
		t.Error("Nop notifier should never notify")
	}

	err := notifier.NotifyRepeatedAuthFailure("client", 10, "ip")
	if err != nil {
		t.Errorf("Nop notifier should not return error, got %v", err)
	}
}

// mockEmailSender for testing
type mockEmailSender struct {
	sentEmails []mockEmail
}

type mockEmail struct {
	to      string
	subject string
	body    string
}

func (m *mockEmailSender) SendEmail(to, subject, body string) error {
	m.sentEmails = append(m.sentEmails, mockEmail{
		to:      to,
		subject: subject,
		body:    body,
	})
	return nil
}
