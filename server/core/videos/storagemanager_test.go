package videos

import (
	"context"
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/db"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/notifications"
)

// mockStorageNotifier is a test implementation of StorageNotifier
type mockStorageNotifier struct {
	shouldWarnThreshold float64
	capacityWarnings    []capacityNotification
	capacityReached     []capacityNotification
	shouldWarnCalled    []shouldWarnCall
}

type capacityNotification struct {
	clientID       string
	usedMegaBytes  int64
	totalMegaBytes int64
}

type shouldWarnCall struct {
	usedMegaBytes  int64
	totalMegaBytes int64
	result         bool
}

func newMockStorageNotifier(shouldWarnThreshold float64) *mockStorageNotifier {
	return &mockStorageNotifier{
		shouldWarnThreshold: shouldWarnThreshold,
		capacityWarnings:    make([]capacityNotification, 0),
		capacityReached:     make([]capacityNotification, 0),
		shouldWarnCalled:    make([]shouldWarnCall, 0),
	}
}

func (m *mockStorageNotifier) NotifyCapacityReached(clientID string, usedMegaBytes int64, totalMegaBytes int64) error {
	m.capacityReached = append(m.capacityReached, capacityNotification{
		clientID:       clientID,
		usedMegaBytes:  usedMegaBytes,
		totalMegaBytes: totalMegaBytes,
	})
	return nil
}

func (m *mockStorageNotifier) NotifyCapacityWarning(clientID string, usedMegaBytes int64, totalMegaBytes int64) error {
	m.capacityWarnings = append(m.capacityWarnings, capacityNotification{
		clientID:       clientID,
		usedMegaBytes:  usedMegaBytes,
		totalMegaBytes: totalMegaBytes,
	})
	return nil
}

func (m *mockStorageNotifier) ShouldWarn(usedMegaBytes int64, totalMegaBytes int64) bool {
	result := float64(usedMegaBytes)/float64(totalMegaBytes) >= m.shouldWarnThreshold
	m.shouldWarnCalled = append(m.shouldWarnCalled, shouldWarnCall{
		usedMegaBytes:  usedMegaBytes,
		totalMegaBytes: totalMegaBytes,
		result:         result,
	})
	return result
}

func setupStorageManagerTest(t *testing.T) (*storageManager, *SQLiteClipRepository, *clients.SQLiteClientRepository, *mockStorageNotifier, func()) {
	// Create in-memory database
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	// Create repositories
	clipRepo, err := NewSQLiteClipRepository(testDB)
	if err != nil {
		testDB.Close()
		t.Fatalf("Failed to create clip repository: %v", err)
	}

	clientRepo, err := clients.NewSQLiteClientRepository(testDB)
	if err != nil {
		testDB.Close()
		t.Fatalf("Failed to create client repository: %v", err)
	}

	// Create mock notifier (80% threshold)
	notifier := newMockStorageNotifier(0.8)

	// Create storage manager
	sm := &storageManager{
		logger:     logging.NopLogger,
		clipRepo:   clipRepo,
		clientRepo: clientRepo,
		notifier:   notifier,
	}

	cleanup := func() {
		testDB.Close()
	}

	return sm, clipRepo, clientRepo, notifier, cleanup
}

func createTestClientForStorage(id string, storageLimitMB int) *clients.Client {
	now := time.Now().UTC()
	return &clients.Client{
		ID:                    id,
		SecretHash:            "hashedSecret123",
		SecretSalt:            "saltValue456",
		CreatedAt:             now,
		UpdatedAt:             now,
		EncryptedMek:          "encryptedMekValue789",
		KeyDerivationSalt:     "keyDerivationSalt012",
		StorageLimitMegabytes: storageLimitMB,
		ClipDurationSeconds:   30,
		MotionOnly:            false,
		Grayscale:             false,
		DownscaleResolution:   "",
	}
}

func createTestClipForStorage(id, clientID string, videoSizeBytes int) *Clip {
	now := time.Now().UTC()
	videoData := make([]byte, videoSizeBytes)
	for i := range videoData {
		videoData[i] = byte(i % 256)
	}

	return &Clip{
		ID:                 id,
		ClientID:           clientID,
		Title:              "Test Video Clip",
		TimeStamp:          now,
		Duration:           time.Duration(30 * time.Second),
		HasMotion:          true,
		EncryptedVideo:     videoData,
		VideoWidth:         1920,
		VideoHeight:        1080,
		VideoMimeType:      "video/mp4",
		EncryptedThumbnail: []byte("encrypted-thumbnail-data"),
		ThumbnailWidth:     320,
		ThumbnailHeight:    240,
		ThumbnailMimeType:  "image/jpeg",
	}
}

func TestStorageManager_StoreClip_UnlimitedStorage(t *testing.T) {
	sm, _, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with unlimited storage (0 or negative limit)
	client := createTestClientForStorage("client-unlimited", 0)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create a large clip
	clip := createTestClipForStorage("clip-1", "client-unlimited", 5*1024*1024) // 5MB

	err = sm.StoreClip(ctx, clip)
	if err != nil {
		t.Fatalf("Expected no error for unlimited storage, got: %v", err)
	}

	// Verify no notifications were sent
	if len(notifier.capacityWarnings) > 0 {
		t.Errorf("Expected no capacity warnings, got %d", len(notifier.capacityWarnings))
	}
	if len(notifier.capacityReached) > 0 {
		t.Errorf("Expected no capacity reached notifications, got %d", len(notifier.capacityReached))
	}
}

func TestStorageManager_StoreClip_WithinLimits(t *testing.T) {
	sm, _, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 10MB storage limit
	client := createTestClientForStorage("client-limited", 10)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create a 2MB clip (well within limits)
	clip := createTestClipForStorage("clip-1", "client-limited", 2*1024*1024) // 2MB

	err = sm.StoreClip(ctx, clip)
	if err != nil {
		t.Fatalf("Expected no error for storage within limits, got: %v", err)
	}

	// Verify no capacity reached notifications
	if len(notifier.capacityReached) > 0 {
		t.Errorf("Expected no capacity reached notifications, got %d", len(notifier.capacityReached))
	}
}

func TestStorageManager_StoreClip_WarningThreshold(t *testing.T) {
	sm, clipRepo, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 10MB storage limit
	client := createTestClientForStorage("client-warning", 10)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add existing clips totaling 8MB (80% usage - exactly at warning threshold)
	for i := 0; i < 8; i++ {
		existingClip := createTestClipForStorage(string(rune('a'+i)), "client-warning", 1*1024*1024) // 1MB each
		existingClip.TimeStamp = time.Now().UTC().Add(-time.Duration(i) * time.Hour)                 // Different timestamps
		err = clipRepo.Add(ctx, existingClip)
		if err != nil {
			t.Fatalf("Failed to add existing clip: %v", err)
		}
	}

	// Create a 1MB clip that will keep usage at 9MB (90% usage, no capacity exceeded)
	clip := createTestClipForStorage("clip-new", "client-warning", 1*1024*1024) // 1MB

	err = sm.StoreClip(ctx, clip)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify warning was sent (current usage of 8MB is at 80% threshold)
	if len(notifier.capacityWarnings) != 1 {
		t.Errorf("Expected 1 capacity warning, got %d", len(notifier.capacityWarnings))
	} else {
		warning := notifier.capacityWarnings[0]
		if warning.clientID != "client-warning" {
			t.Errorf("Expected client ID 'client-warning', got '%s'", warning.clientID)
		}
		if warning.usedMegaBytes != 8 { // Current usage before adding the new clip
			t.Errorf("Expected used MB to be 8, got %d", warning.usedMegaBytes)
		}
		if warning.totalMegaBytes != 10 {
			t.Errorf("Expected total MB to be 10, got %d", warning.totalMegaBytes)
		}
	}

	// Verify no capacity reached notifications
	if len(notifier.capacityReached) > 0 {
		t.Errorf("Expected no capacity reached notifications, got %d", len(notifier.capacityReached))
	}
}

func TestStorageManager_StoreClip_CapacityExceeded_DeletesOldestClips(t *testing.T) {
	sm, clipRepo, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 5MB storage limit
	client := createTestClientForStorage("client-exceeded", 5)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add existing clips totaling 4MB
	oldestTime := time.Now().UTC().Add(-4 * time.Hour)
	secondOldestTime := time.Now().UTC().Add(-3 * time.Hour)
	newestTime := time.Now().UTC().Add(-1 * time.Hour)

	oldestClip := createTestClipForStorage("oldest-clip", "client-exceeded", 2*1024*1024) // 2MB
	oldestClip.TimeStamp = oldestTime
	err = clipRepo.Add(ctx, oldestClip)
	if err != nil {
		t.Fatalf("Failed to add oldest clip: %v", err)
	}

	secondOldestClip := createTestClipForStorage("second-oldest-clip", "client-exceeded", 1*1024*1024) // 1MB
	secondOldestClip.TimeStamp = secondOldestTime
	err = clipRepo.Add(ctx, secondOldestClip)
	if err != nil {
		t.Fatalf("Failed to add second oldest clip: %v", err)
	}

	newestClip := createTestClipForStorage("newest-clip", "client-exceeded", 1*1024*1024) // 1MB
	newestClip.TimeStamp = newestTime
	err = clipRepo.Add(ctx, newestClip)
	if err != nil {
		t.Fatalf("Failed to add newest clip: %v", err)
	}

	// Try to add a 2MB clip (total would be 6MB, exceeding 5MB limit)
	// After deleting oldest (2MB), we'd have 2MB + 2MB = 4MB, which is within limit
	newClip := createTestClipForStorage("new-large-clip", "client-exceeded", 2*1024*1024) // 2MB

	err = sm.StoreClip(ctx, newClip)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify capacity reached notification was sent
	if len(notifier.capacityReached) != 1 {
		t.Errorf("Expected 1 capacity reached notification, got %d", len(notifier.capacityReached))
	}

	// Verify the oldest clip was deleted
	retrievedOldest, err := clipRepo.GetByID(ctx, "oldest-clip")
	if err != nil {
		t.Fatalf("Error checking for oldest clip: %v", err)
	}
	if retrievedOldest != nil {
		t.Error("Expected oldest clip to be deleted, but it still exists")
	}

	// Verify the second oldest clip still exists (only needed to delete oldest)
	retrievedSecondOldest, err := clipRepo.GetByID(ctx, "second-oldest-clip")
	if err != nil {
		t.Fatalf("Error checking for second oldest clip: %v", err)
	}
	if retrievedSecondOldest == nil {
		t.Error("Expected second oldest clip to still exist, but it was deleted")
	}

	// Verify the newest clip still exists
	retrievedNewest, err := clipRepo.GetByID(ctx, "newest-clip")
	if err != nil {
		t.Fatalf("Error checking for newest clip: %v", err)
	}
	if retrievedNewest == nil {
		t.Error("Expected newest clip to still exist, but it was deleted")
	}

	// Verify the new clip was added
	retrievedNew, err := clipRepo.GetByID(ctx, "new-large-clip")
	if err != nil {
		t.Fatalf("Error checking for new clip: %v", err)
	}
	if retrievedNew == nil {
		t.Error("Expected new clip to be added, but it doesn't exist")
	}
}

func TestStorageManager_StoreClip_CapacityExceeded_NoOldClipsToDelete(t *testing.T) {
	sm, _, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 2MB storage limit
	client := createTestClientForStorage("client-no-old", 2)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Try to add a 3MB clip (exceeds limit and no existing clips to delete)
	newClip := createTestClipForStorage("large-clip", "client-no-old", 3*1024*1024) // 3MB

	err = sm.StoreClip(ctx, newClip)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify capacity reached notification was sent
	if len(notifier.capacityReached) != 1 {
		t.Errorf("Expected 1 capacity reached notification, got %d", len(notifier.capacityReached))
	}

	// Verify the clip was still added (even though it exceeds capacity)
	retrievedClip, err := sm.clipRepo.GetByID(ctx, "large-clip")
	if err != nil {
		t.Fatalf("Error checking for clip: %v", err)
	}
	if retrievedClip == nil {
		t.Error("Expected clip to be added even when exceeding capacity with no old clips to delete")
	}
}

func TestStorageManager_StoreClip_ClientNotFound(t *testing.T) {
	sm, _, _, _, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Try to store clip for non-existent client
	clip := createTestClipForStorage("clip-1", "non-existent-client", 1*1024*1024)

	err := sm.StoreClip(ctx, clip)
	if err == nil {
		t.Fatal("Expected error for non-existent client, got nil")
	}
}

func TestStorageManager_StoreClip_WarningNotSentWhenCapacityExceeded(t *testing.T) {
	sm, clipRepo, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 5MB storage limit
	client := createTestClientForStorage("client-test", 5)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add existing clips totaling 4MB (80% usage - at warning threshold)
	for i := 0; i < 4; i++ {
		existingClip := createTestClipForStorage(string(rune('a'+i)), "client-test", 1*1024*1024) // 1MB each
		existingClip.TimeStamp = time.Now().UTC().Add(-time.Duration(i) * time.Hour)
		err = clipRepo.Add(ctx, existingClip)
		if err != nil {
			t.Fatalf("Failed to add existing clip: %v", err)
		}
	}

	// Try to add a 2MB clip (will exceed capacity)
	clip := createTestClipForStorage("clip-new", "client-test", 2*1024*1024) // 2MB

	err = sm.StoreClip(ctx, clip)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify no warning was sent (because capacity was exceeded)
	if len(notifier.capacityWarnings) > 0 {
		t.Errorf("Expected no capacity warnings when capacity exceeded, got %d", len(notifier.capacityWarnings))
	}

	// Verify capacity reached notification was sent
	if len(notifier.capacityReached) != 1 {
		t.Errorf("Expected 1 capacity reached notification, got %d", len(notifier.capacityReached))
	}
}

func TestStorageManager_StoreClip_MultipleClipDeletionLoop(t *testing.T) {
	sm, clipRepo, clientRepo, _, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 10MB storage limit
	client := createTestClientForStorage("client-multi-delete", 10)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add 8 clips of 1MB each (8MB total)
	baseTime := time.Now().UTC().Add(-10 * time.Hour)
	for i := 0; i < 8; i++ {
		clip := createTestClipForStorage(string(rune('a'+i)), "client-multi-delete", 1*1024*1024) // 1MB each
		clip.TimeStamp = baseTime.Add(time.Duration(i) * time.Hour)                               // Spread them out in time
		err = clipRepo.Add(ctx, clip)
		if err != nil {
			t.Fatalf("Failed to add clip %d: %v", i, err)
		}
	}

	// Try to add a 5MB clip (total would be 13MB, need to delete 3+ MB worth of clips)
	newClip := createTestClipForStorage("new-big-clip", "client-multi-delete", 5*1024*1024) // 5MB

	err = sm.StoreClip(ctx, newClip)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the new clip was added
	retrievedNew, err := clipRepo.GetByID(ctx, "new-big-clip")
	if err != nil {
		t.Fatalf("Error checking for new clip: %v", err)
	}
	if retrievedNew == nil {
		t.Error("Expected new clip to be added")
	}

	// Verify at least 3 old clips were deleted (to make room for the 5MB clip)
	deletedCount := 0
	for i := 0; i < 8; i++ {
		clipID := string(rune('a' + i))
		retrieved, err := clipRepo.GetByID(ctx, clipID)
		if err != nil {
			t.Fatalf("Error checking for clip %s: %v", clipID, err)
		}
		if retrieved == nil {
			deletedCount++
		}
	}

	if deletedCount < 3 {
		t.Errorf("Expected at least 3 clips to be deleted to make room, got %d", deletedCount)
	}

	// Verify the remaining storage is within limits
	totalUsage, err := clipRepo.GetTotalStorageUsage(ctx, "client-multi-delete")
	if err != nil {
		t.Fatalf("Failed to get total storage usage: %v", err)
	}
	totalUsageMB := totalUsage / (1024 * 1024)
	if totalUsageMB > 10 {
		t.Errorf("Expected total storage usage to be <= 10MB, got %dMB", totalUsageMB)
	}
}

func TestNewStorageManager_WithNilLogger(t *testing.T) {
	_, clipRepo, clientRepo, _, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	sm := NewStorageManager(nil, clipRepo, clientRepo, nil)

	// Verify that the storage manager was created successfully with NopLogger and NopStorageNotifier
	smImpl := sm.(*storageManager)
	if smImpl.logger != logging.NopLogger {
		t.Error("Expected NopLogger to be used when nil logger is provided")
	}
	if smImpl.notifier != notifications.NopStorageNotifier {
		t.Error("Expected NopStorageNotifier to be used when nil notifier is provided")
	}
}

func TestStorageManager_StoreClip_ExactCapacityLimit(t *testing.T) {
	sm, _, clientRepo, notifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with 5MB storage limit
	client := createTestClientForStorage("client-exact", 5)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Try to add a clip that exactly matches the limit
	clip := createTestClipForStorage("exact-clip", "client-exact", 5*1024*1024) // 5MB

	err = sm.StoreClip(ctx, clip)
	if err != nil {
		t.Fatalf("Expected no error for exact capacity limit, got: %v", err)
	}

	// Since the new clip exactly matches the limit, there should be no capacity exceeded
	if len(notifier.capacityReached) > 0 {
		t.Errorf("Expected no capacity reached notifications for exact limit, got %d", len(notifier.capacityReached))
	}

	// Verify the clip was added
	retrievedClip, err := sm.clipRepo.GetByID(ctx, "exact-clip")
	if err != nil {
		t.Fatalf("Error checking for clip: %v", err)
	}
	if retrievedClip == nil {
		t.Error("Expected clip to be added at exact capacity limit")
	}
}
