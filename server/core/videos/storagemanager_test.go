package videos

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/db"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/notifications"

	_ "github.com/mattn/go-sqlite3"
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

// mockMotionNotifier is a test implementation of MotionNotifier
type mockMotionNotifier struct {
	motionNotifications []motionNotification
}

type motionNotification struct {
	clientID  string
	clipTitle string
	timestamp time.Time
}

func newMockMotionNotifier() *mockMotionNotifier {
	return &mockMotionNotifier{
		motionNotifications: make([]motionNotification, 0),
	}
}

func (m *mockMotionNotifier) NotifyMotionDetected(clientID string, clipTitle string, timestamp time.Time) error {
	m.motionNotifications = append(m.motionNotifications, motionNotification{
		clientID:  clientID,
		clipTitle: clipTitle,
		timestamp: timestamp,
	})
	return nil
}

func setupStorageManagerTest(t *testing.T) (*storageManager, *SQLiteClipRepository, *clients.SQLiteClientRepository, *mockStorageNotifier, *mockMotionNotifier, func()) {
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

	// Create mock notifiers
	storageNotifier := newMockStorageNotifier(0.8) // 80% threshold
	motionNotifier := newMockMotionNotifier()

	// Create storage manager
	sm := &storageManager{
		logger:               logging.NopLogger,
		clipRepo:             clipRepo,
		clientRepo:           clientRepo,
		notifier:             storageNotifier,
		motionNotifier:       motionNotifier,
		clientStorageMutexes: sync.Map{},
	}

	cleanup := func() {
		testDB.Close()
	}

	return sm, clipRepo, clientRepo, storageNotifier, motionNotifier, cleanup
}

// setupConcurrencyTest creates a test environment with SQLite optimizations for concurrency testing
func setupConcurrencyTest(t *testing.T) (*storageManager, *SQLiteClipRepository, *clients.SQLiteClientRepository, *mockStorageNotifier, *mockMotionNotifier, func()) {
	// Create in-memory database with SQLite optimizations for concurrency
	// Use shared cache to allow multiple connections to the same in-memory database
	dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared&_journal_mode=WAL&_busy_timeout=30000&_synchronous=NORMAL&_cache_size=10000")
	if err != nil {
		t.Fatalf("Failed to create optimized in-memory database: %v", err)
	}

	// Configure connection pool for better concurrency (same as production)
	dbConn.SetMaxOpenConns(10)                  // Allow up to 10 concurrent connections
	dbConn.SetMaxIdleConns(5)                   // Keep 5 idle connections
	dbConn.SetConnMaxLifetime(30 * time.Minute) // Rotate connections every 30 minutes

	// Create repositories
	clipRepo, err := NewSQLiteClipRepository(dbConn)
	if err != nil {
		dbConn.Close()
		t.Fatalf("Failed to create clip repository: %v", err)
	}

	clientRepo, err := clients.NewSQLiteClientRepository(dbConn)
	if err != nil {
		dbConn.Close()
		t.Fatalf("Failed to create client repository: %v", err)
	}

	// Create mock notifiers
	storageNotifier := newMockStorageNotifier(0.8) // 80% threshold
	motionNotifier := newMockMotionNotifier()

	// Create storage manager
	sm := &storageManager{
		logger:               logging.NopLogger,
		clipRepo:             clipRepo,
		clientRepo:           clientRepo,
		notifier:             storageNotifier,
		motionNotifier:       motionNotifier,
		clientStorageMutexes: sync.Map{},
	}

	cleanup := func() {
		dbConn.Close()
	}

	return sm, clipRepo, clientRepo, storageNotifier, motionNotifier, cleanup
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
	sm, _, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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
	sm, _, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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
	sm, clipRepo, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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
	sm, clipRepo, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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
	sm, _, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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
	sm, _, _, _, _, cleanup := setupStorageManagerTest(t)
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
	sm, clipRepo, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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
	sm, clipRepo, clientRepo, _, _, cleanup := setupStorageManagerTest(t)
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
	_, clipRepo, clientRepo, _, _, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	sm := NewStorageManager(nil, clipRepo, clientRepo, nil, nil)

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
	sm, _, clientRepo, notifier, _, cleanup := setupStorageManagerTest(t)
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

func TestStorageManager_ConcurrentUploads(t *testing.T) {
	sm, clipRepo, clientRepo, notifier, _, cleanup := setupConcurrencyTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a client with limited storage (10MB)
	client := createTestClientForStorage("concurrent-client", 10)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Add some existing clips to use up most of the space (8MB)
	existingClip := createTestClipForStorage("existing-clip", "concurrent-client", 8*1024*1024) // 8MB
	err = sm.StoreClip(ctx, existingClip)
	if err != nil {
		t.Fatalf("Failed to store existing clip: %v", err)
	}

	// Now we have 2MB left. Let's try to upload 5 clips of 1MB each concurrently
	// Only 2 should succeed, the rest should trigger cleanup
	numGoroutines := 5
	clipSize := 1 * 1024 * 1024 // 1MB each

	// Channels to collect results
	results := make(chan error, numGoroutines)
	clipIDs := make(chan string, numGoroutines)

	// Launch concurrent uploads
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			clipID := fmt.Sprintf("concurrent-clip-%d", index)
			clip := createTestClipForStorage(clipID, "concurrent-client", clipSize)

			err := sm.StoreClip(ctx, clip)
			results <- err
			if err == nil {
				clipIDs <- clipID
			}
		}(i)
	}

	// Collect results
	var successCount int
	var errorCount int
	var successfulClipIDs []string

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err != nil {
			errorCount++
			t.Logf("Upload %d failed: %v", i, err)
		} else {
			successCount++
		}
	}

	// Collect successful clip IDs
	close(clipIDs)
	for clipID := range clipIDs {
		successfulClipIDs = append(successfulClipIDs, clipID)
	}

	t.Logf("Concurrent upload results: %d successes, %d errors", successCount, errorCount)

	// Verify that we didn't exceed storage limits
	totalUsage, err := clipRepo.GetTotalStorageUsage(ctx, "concurrent-client")
	if err != nil {
		t.Fatalf("Failed to get total storage usage: %v", err)
	}

	totalUsageMB := totalUsage / (1024 * 1024)
	t.Logf("Total storage usage after concurrent uploads: %d MB", totalUsageMB)

	// Should not exceed the 10MB limit
	if totalUsageMB > 10 {
		t.Errorf("Storage limit exceeded: %d MB > 10 MB", totalUsageMB)
	}

	// At least some uploads should have succeeded
	if successCount == 0 {
		t.Error("No uploads succeeded - this suggests a serious concurrency issue")
	}

	// Verify that capacity reached notifications were sent
	if len(notifier.capacityReached) == 0 {
		t.Error("Expected capacity reached notifications to be sent")
	}

	// Verify all successful clips are actually stored and can be retrieved
	for _, clipID := range successfulClipIDs {
		retrievedClip, err := clipRepo.GetByID(ctx, clipID)
		if err != nil {
			t.Errorf("Failed to retrieve successfully stored clip %s: %v", clipID, err)
		}
		if retrievedClip == nil {
			t.Errorf("Successfully stored clip %s not found in database", clipID)
		}
	}
}

func TestStorageManager_ConcurrentUploadsStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	sm, clipRepo, clientRepo, _, _, cleanup := setupConcurrencyTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a client with unlimited storage for stress testing
	client := createTestClientForStorage("stress-client", 0) // 0 = unlimited
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Perform many concurrent uploads to test for database lock issues
	numGoroutines := 20
	uploadsPerGoroutine := 5
	clipSize := 1024 // 1KB clips

	results := make(chan error, numGoroutines*uploadsPerGoroutine)

	// Launch many concurrent uploads
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for u := 0; u < uploadsPerGoroutine; u++ {
				clipID := fmt.Sprintf("stress-clip-%d-%d", goroutineID, u)
				clip := createTestClipForStorage(clipID, "stress-client", clipSize)

				err := sm.StoreClip(ctx, clip)
				results <- err
			}
		}(g)
	}

	// Collect results
	var successCount int
	var errorCount int
	totalOperations := numGoroutines * uploadsPerGoroutine

	for i := 0; i < totalOperations; i++ {
		err := <-results
		if err != nil {
			errorCount++
			t.Logf("Stress test upload failed: %v", err)
		} else {
			successCount++
		}
	}

	t.Logf("Stress test results: %d successes, %d errors out of %d total operations",
		successCount, errorCount, totalOperations)

	// In a stress test with unlimited storage, most operations should succeed
	successRate := float64(successCount) / float64(totalOperations)
	if successRate < 0.95 { // Allow for some failures, but expect 95%+ success
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate*100)
	}

	// Verify the expected number of clips were stored
	// Note: some clips might have been overwritten if they had the same timestamp
	// so we just verify we have a reasonable number stored
	clips, totalCount, err := clipRepo.Query(ctx, ClipQuery{ClientID: "stress-client"})
	if err != nil {
		t.Fatalf("Failed to query clips: %v", err)
	}

	t.Logf("Total clips stored after stress test: %d", totalCount)
	if totalCount == 0 {
		t.Error("No clips were stored during stress test")
	}

	// Verify we can retrieve all stored clips without errors
	for _, clip := range clips {
		retrievedClip, err := clipRepo.GetByID(ctx, clip.ID)
		if err != nil {
			t.Errorf("Failed to retrieve clip %s: %v", clip.ID, err)
		}
		if retrievedClip == nil {
			t.Errorf("Clip %s not found", clip.ID)
		}
	}
}

func TestStorageManager_ConcurrentStorageLimitRaceCondition(t *testing.T) {
	sm, clipRepo, clientRepo, _, _, cleanup := setupConcurrencyTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a client with exactly 5MB storage limit
	client := createTestClientForStorage("race-client", 5)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Add existing clips to use up 3MB (leaving 2MB free)
	existingClip := createTestClipForStorage("existing-clip", "race-client", 3*1024*1024) // 3MB
	err = sm.StoreClip(ctx, existingClip)
	if err != nil {
		t.Fatalf("Failed to store existing clip: %v", err)
	}

	// Now try to upload 10 clips of 1MB each concurrently
	// In the race condition scenario, multiple might read "3MB used" and think they can add 1MB
	// But only 2 should actually succeed without causing storage limit violations
	numGoroutines := 10
	clipSize := 1 * 1024 * 1024 // 1MB each

	results := make(chan error, numGoroutines)
	successfulClips := make(chan string, numGoroutines)

	// Launch concurrent uploads
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			clipID := fmt.Sprintf("race-clip-%d", index)
			clip := createTestClipForStorage(clipID, "race-client", clipSize)
			// Add slight time variation to make clips more distinct
			clip.TimeStamp = time.Now().UTC().Add(time.Duration(index) * time.Millisecond)

			err := sm.StoreClip(ctx, clip)
			results <- err
			if err == nil {
				successfulClips <- clipID
			}
		}(i)
	}

	// Collect results
	var successCount int
	var errorCount int
	var storedClipIDs []string

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// Collect successful clip IDs
	close(successfulClips)
	for clipID := range successfulClips {
		storedClipIDs = append(storedClipIDs, clipID)
	}

	t.Logf("Race condition test results: %d successes, %d errors", successCount, errorCount)

	// Check final storage usage - this is the critical test
	totalUsage, err := clipRepo.GetTotalStorageUsage(ctx, "race-client")
	if err != nil {
		t.Fatalf("Failed to get total storage usage: %v", err)
	}

	totalUsageMB := totalUsage / (1024 * 1024)
	t.Logf("Final storage usage: %d MB (limit: 5 MB)", totalUsageMB)

	// This is the key assertion - we should never exceed the storage limit
	// Even if there are race conditions in our logic, the storage should not exceed 5MB
	if totalUsageMB > 5 {
		t.Errorf("CRITICAL: Storage limit exceeded due to race condition: %d MB > 5 MB", totalUsageMB)

		// Additional debugging information
		t.Logf("Stored clip IDs: %v", storedClipIDs)

		// Check what clips are actually stored
		allClips, _, err := clipRepo.Query(ctx, ClipQuery{ClientID: "race-client"})
		if err != nil {
			t.Logf("Failed to query all clips for debugging: %v", err)
		} else {
			t.Logf("All clips in database:")
			for _, clip := range allClips {
				clipSizeMB := int64(len(clip.EncryptedVideo)) / (1024 * 1024)
				t.Logf("  - %s: %d MB at %v", clip.ID, clipSizeMB, clip.TimeStamp)
			}
		}
	}

	// At least some uploads should succeed (we had 2MB free space)
	if successCount == 0 {
		t.Error("No uploads succeeded - this suggests all operations are failing")
	}

	// Verify that clips currently in database are retrievable
	// Note: Some successfully stored clips might have been deleted by cleanup from other concurrent operations
	currentClips, _, err := clipRepo.Query(ctx, ClipQuery{ClientID: "race-client"})
	if err != nil {
		t.Fatalf("Failed to query current clips: %v", err)
	}

	t.Logf("Current clips remaining in database: %d", len(currentClips))
	for _, clip := range currentClips {
		retrievedClip, err := clipRepo.GetByID(ctx, clip.ID)
		if err != nil {
			t.Errorf("Failed to retrieve current clip %s: %v", clip.ID, err)
		}
		if retrievedClip == nil {
			t.Errorf("Current clip %s not found in database", clip.ID)
		}
	}

	// The key test: verify that concurrent operations didn't cause storage limit violations
	// This is the critical assertion that proves the race condition is fixed
	if totalUsageMB > 5 {
		t.Errorf("CRITICAL: Storage limit exceeded due to race condition: %d MB > 5 MB", totalUsageMB)
	} else {
		t.Logf("SUCCESS: Storage limit respected despite %d concurrent operations", numGoroutines)
	}

	// Note: It's normal for some "successful" clips to be missing from the database
	// because cleanup operations from other concurrent uploads may have deleted them.
	// This is correct behavior - the clip was successfully stored, but later cleaned up.
}

func TestStorageManager_MotionNotification(t *testing.T) {
	sm, _, clientRepo, _, motionNotifier, cleanup := setupStorageManagerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create client with unlimited storage
	client := createTestClientForStorage("client-motion", 0)
	err := clientRepo.Create(ctx, client)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create clip with motion
	clipWithMotion := createTestClipForStorage("clip-with-motion", "client-motion", 1*1024*1024) // 1MB
	clipWithMotion.HasMotion = true
	clipWithMotion.Title = "2024-08-14T10-30-00_30s_motion.mp4"

	// Store the clip
	err = sm.StoreClip(ctx, clipWithMotion)
	if err != nil {
		t.Fatalf("Failed to store clip with motion: %v", err)
	}

	// Verify motion notification was sent
	if len(motionNotifier.motionNotifications) != 1 {
		t.Errorf("Expected 1 motion notification, got %d", len(motionNotifier.motionNotifications))
	} else {
		notification := motionNotifier.motionNotifications[0]
		if notification.clientID != "client-motion" {
			t.Errorf("Expected client ID 'client-motion', got '%s'", notification.clientID)
		}
		if notification.clipTitle != clipWithMotion.Title {
			t.Errorf("Expected clip title '%s', got '%s'", clipWithMotion.Title, notification.clipTitle)
		}
		if !notification.timestamp.Equal(clipWithMotion.TimeStamp) {
			t.Errorf("Expected timestamp %v, got %v", clipWithMotion.TimeStamp, notification.timestamp)
		}
	}

	// Create clip without motion
	clipWithoutMotion := createTestClipForStorage("clip-without-motion", "client-motion", 1*1024*1024) // 1MB
	clipWithoutMotion.HasMotion = false
	clipWithoutMotion.Title = "2024-08-14T10-35-00_30s_nomotion.mp4"

	// Store the clip
	err = sm.StoreClip(ctx, clipWithoutMotion)
	if err != nil {
		t.Fatalf("Failed to store clip without motion: %v", err)
	}

	// Verify no additional motion notification was sent
	if len(motionNotifier.motionNotifications) != 1 {
		t.Errorf("Expected still 1 motion notification (no new ones), got %d", len(motionNotifier.motionNotifications))
	}
}
