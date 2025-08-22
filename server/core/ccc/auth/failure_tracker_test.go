package auth

import (
	"testing"
	"time"
)

func TestMemoryFailureTracker_RecordFailure(t *testing.T) {
	settings := AutoDisableSettings{
		Threshold:  5,
		TimeWindow: time.Hour,
	}

	tracker := NewMemoryFailureTracker(settings)
	clientID := "test-client"
	clientIP := "192.168.1.100"
	now := time.Now()

	// Test recording multiple failures
	count1 := tracker.RecordFailure(clientID, clientIP, now)
	if count1 != 1 {
		t.Errorf("Expected failure count 1, got %d", count1)
	}

	count2 := tracker.RecordFailure(clientID, clientIP, now.Add(1*time.Minute))
	if count2 != 2 {
		t.Errorf("Expected failure count 2, got %d", count2)
	}

	count3 := tracker.RecordFailure(clientID, clientIP, now.Add(2*time.Minute))
	if count3 != 3 {
		t.Errorf("Expected failure count 3, got %d", count3)
	}

	// Test failures from different clients don't interfere
	otherClientCount := tracker.RecordFailure("other-client", clientIP, now.Add(3*time.Minute))
	if otherClientCount != 1 {
		t.Errorf("Expected failure count 1 for other client, got %d", otherClientCount)
	}

	// Original client count should remain the same
	count4 := tracker.RecordFailure(clientID, clientIP, now.Add(4*time.Minute))
	if count4 != 4 {
		t.Errorf("Expected failure count 4 for original client, got %d", count4)
	}
}

func TestMemoryFailureTracker_TimeWindow(t *testing.T) {
	settings := AutoDisableSettings{
		Threshold:  5,
		TimeWindow: 10 * time.Minute,
	}

	tracker := NewMemoryFailureTracker(settings)
	clientID := "test-client"
	clientIP := "192.168.1.100"
	now := time.Now()

	// Record failures within time window
	tracker.RecordFailure(clientID, clientIP, now)
	tracker.RecordFailure(clientID, clientIP, now.Add(2*time.Minute))
	tracker.RecordFailure(clientID, clientIP, now.Add(5*time.Minute))

	// This failure should only count the ones within the time window
	count := tracker.RecordFailure(clientID, clientIP, now.Add(15*time.Minute))

	// The cutoff time is (now+15min) - 10min = now+5min
	// So failures at or after now+5min should be counted: now+5min and now+15min = 2 total
	if count != 2 {
		t.Errorf("Expected failure count 2 (within time window), got %d", count)
	}
}

func TestMemoryFailureTracker_ShouldAutoDisable(t *testing.T) {
	settings := AutoDisableSettings{
		Threshold:  3,
		TimeWindow: time.Hour,
	}

	tracker := NewMemoryFailureTracker(settings)

	// Should not auto-disable below threshold
	if tracker.ShouldAutoDisable(2) {
		t.Error("Should not auto-disable with failure count below threshold")
	}

	// Should auto-disable at threshold
	if !tracker.ShouldAutoDisable(3) {
		t.Error("Should auto-disable with failure count at threshold")
	}

	// Should auto-disable above threshold
	if !tracker.ShouldAutoDisable(5) {
		t.Error("Should auto-disable with failure count above threshold")
	}

	// Test with auto-disable disabled (threshold = 0)
	settingsDisabled := AutoDisableSettings{
		Threshold:  0,
		TimeWindow: time.Hour,
	}
	trackerDisabled := NewMemoryFailureTracker(settingsDisabled)

	if trackerDisabled.ShouldAutoDisable(100) {
		t.Error("Should not auto-disable when threshold is 0")
	}
}

func TestNopFailureTracker(t *testing.T) {
	tracker := NopFailureTracker

	// Should always return 0 for failure count
	count := tracker.RecordFailure("client", "ip", time.Now())
	if count != 0 {
		t.Errorf("Expected 0 failure count, got %d", count)
	}

	// Should never auto-disable
	if tracker.ShouldAutoDisable(100) {
		t.Error("Nop tracker should never auto-disable")
	}
}
