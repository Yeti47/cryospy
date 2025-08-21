package streaming

import (
	"fmt"
	"testing"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

func TestNewNormalizedClipCache(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	stats := cache.Stats()
	if stats.MaxSize != 1024 {
		t.Errorf("Expected max size 1024, got %d", stats.MaxSize)
	}
	if stats.TotalSize != 0 {
		t.Errorf("Expected initial size 0, got %d", stats.TotalSize)
	}
	if stats.EntryCount != 0 {
		t.Errorf("Expected initial entry count 0, got %d", stats.EntryCount)
	}
}

func TestNewNormalizedClipCacheInvalidSize(t *testing.T) {
	cache := NewNormalizedClipCache(-1, logging.NopLogger)
	stats := cache.Stats()
	expectedSize := int64(100 * 1024 * 1024) // 100MB default
	if stats.MaxSize != expectedSize {
		t.Errorf("Expected default size %d, got %d", expectedSize, stats.MaxSize)
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	clipID := "test-clip-1"
	data := []byte("test video data")

	// Test cache miss
	_, found := cache.Get(clipID)
	if found {
		t.Error("Expected cache miss for non-existent clip")
	}

	// Set data
	cache.Set(clipID, data)

	// Test cache hit
	retrievedData, found := cache.Get(clipID)
	if !found {
		t.Error("Expected cache hit for existing clip")
	}

	if string(retrievedData) != string(data) {
		t.Errorf("Expected data '%s', got '%s'", string(data), string(retrievedData))
	}

	// Verify stats
	stats := cache.Stats()
	if stats.TotalSize != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), stats.TotalSize)
	}
	if stats.EntryCount != 1 {
		t.Errorf("Expected entry count 1, got %d", stats.EntryCount)
	}
	if stats.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Errorf("Expected miss count 1, got %d", stats.MissCount)
	}
}

func TestCacheUpdate(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	clipID := "test-clip-1"
	originalData := []byte("original data")
	updatedData := []byte("updated data with more content")

	// Set original data
	cache.Set(clipID, originalData)
	stats := cache.Stats()
	originalSize := stats.TotalSize

	// Update with new data
	cache.Set(clipID, updatedData)

	// Verify update
	retrievedData, found := cache.Get(clipID)
	if !found {
		t.Error("Expected cache hit after update")
	}
	if string(retrievedData) != string(updatedData) {
		t.Errorf("Expected updated data '%s', got '%s'", string(updatedData), string(retrievedData))
	}

	// Verify size change
	stats = cache.Stats()
	expectedSize := int64(len(updatedData))
	if stats.TotalSize != expectedSize {
		t.Errorf("Expected size %d after update, got %d", expectedSize, stats.TotalSize)
	}
	if stats.EntryCount != 1 {
		t.Errorf("Expected entry count to remain 1 after update, got %d", stats.EntryCount)
	}

	sizeDiff := stats.TotalSize - originalSize
	expectedDiff := int64(len(updatedData) - len(originalData))
	if sizeDiff != expectedDiff {
		t.Errorf("Expected size difference %d, got %d", expectedDiff, sizeDiff)
	}
}

func TestCacheEviction(t *testing.T) {
	maxSize := int64(50) // Small cache
	cache := NewNormalizedClipCache(maxSize, logging.NopLogger)

	// Add entries that exceed cache size
	cache.Set("clip1", []byte("data1234567890123456789012345")) // 25 bytes
	cache.Set("clip2", []byte("data1234567890123456789012345")) // 25 bytes - total 50, at limit
	cache.Set("clip3", []byte("data12345"))                     // 9 bytes - should trigger eviction

	stats := cache.Stats()
	if stats.TotalSize > maxSize {
		t.Errorf("Cache size %d exceeds max size %d", stats.TotalSize, maxSize)
	}

	// clip1 should be evicted (least recently used)
	_, found := cache.Get("clip1")
	if found {
		t.Error("Expected clip1 to be evicted")
	}

	// clip2 and clip3 should still be present
	_, found = cache.Get("clip2")
	if !found {
		t.Error("Expected clip2 to still be in cache")
	}
	_, found = cache.Get("clip3")
	if !found {
		t.Error("Expected clip3 to still be in cache")
	}

	if stats.EvictionCount == 0 {
		t.Error("Expected at least one eviction")
	}
}

func TestCacheLRUOrdering(t *testing.T) {
	maxSize := int64(60)
	cache := NewNormalizedClipCache(maxSize, logging.NopLogger)

	// Add three entries
	cache.Set("clip1", []byte("12345678901234567890")) // 20 bytes
	cache.Set("clip2", []byte("12345678901234567890")) // 20 bytes
	cache.Set("clip3", []byte("12345678901234567890")) // 20 bytes - total 60, at limit

	// Access clip1 to make it most recently used
	cache.Get("clip1")

	// Add another entry, should evict clip2 (least recently used)
	cache.Set("clip4", []byte("123")) // 3 bytes

	// Verify clip2 was evicted but clip1 remains
	_, found := cache.Get("clip2")
	if found {
		t.Error("Expected clip2 to be evicted")
	}
	_, found = cache.Get("clip1")
	if !found {
		t.Error("Expected clip1 to remain (was accessed recently)")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	clipID := "test-clip-1"
	data := []byte("test data")

	cache.Set(clipID, data)
	cache.Delete(clipID)

	_, found := cache.Get(clipID)
	if found {
		t.Error("Expected clip to be deleted from cache")
	}

	stats := cache.Stats()
	if stats.TotalSize != 0 {
		t.Errorf("Expected size 0 after delete, got %d", stats.TotalSize)
	}
	if stats.EntryCount != 0 {
		t.Errorf("Expected entry count 0 after delete, got %d", stats.EntryCount)
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	// Add multiple entries
	cache.Set("clip1", []byte("data1"))
	cache.Set("clip2", []byte("data2"))
	cache.Set("clip3", []byte("data3"))

	cache.Clear()

	// Verify all entries are gone
	_, found := cache.Get("clip1")
	if found {
		t.Error("Expected clip1 to be cleared")
	}
	_, found = cache.Get("clip2")
	if found {
		t.Error("Expected clip2 to be cleared")
	}
	_, found = cache.Get("clip3")
	if found {
		t.Error("Expected clip3 to be cleared")
	}

	stats := cache.Stats()
	if stats.TotalSize != 0 {
		t.Errorf("Expected size 0 after clear, got %d", stats.TotalSize)
	}
	if stats.EntryCount != 0 {
		t.Errorf("Expected entry count 0 after clear, got %d", stats.EntryCount)
	}
}

func TestCacheEmptyData(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	clipID := "test-clip-1"
	emptyData := []byte{}

	cache.Set(clipID, emptyData)

	// Should not be cached
	_, found := cache.Get(clipID)
	if found {
		t.Error("Expected empty data not to be cached")
	}

	stats := cache.Stats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected entry count 0 for empty data, got %d", stats.EntryCount)
	}
}

func TestCacheDataTooLarge(t *testing.T) {
	maxSize := int64(10)
	cache := NewNormalizedClipCache(maxSize, logging.NopLogger)

	clipID := "test-clip-1"
	largeData := []byte("this data is larger than the cache")

	cache.Set(clipID, largeData)

	// Should not be cached
	_, found := cache.Get(clipID)
	if found {
		t.Error("Expected data too large not to be cached")
	}

	stats := cache.Stats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected entry count 0 for oversized data, got %d", stats.EntryCount)
	}
}

func TestCacheDataIsolation(t *testing.T) {
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	clipID := "test-clip-1"
	originalData := []byte("original data")

	cache.Set(clipID, originalData)

	// Get data and modify it
	retrievedData, _ := cache.Get(clipID)
	copy(retrievedData, "modified!!")

	// Get data again and verify it wasn't modified
	freshData, _ := cache.Get(clipID)
	if string(freshData) != string(originalData) {
		t.Error("Cache data was modified externally - data isolation failed")
	}
}

func TestCacheStats(t *testing.T) {
	maxSize := int64(1000)
	cache := NewNormalizedClipCache(maxSize, logging.NopLogger)

	// Test initial stats
	stats := cache.Stats()
	if stats.MaxSize != maxSize {
		t.Errorf("Expected max size %d, got %d", maxSize, stats.MaxSize)
	}
	if stats.UtilizationPct != 0 {
		t.Errorf("Expected initial utilization 0%%, got %.2f%%", stats.UtilizationPct)
	}

	// Add some data
	cache.Set("clip1", []byte("test data 1234567890")) // 20 bytes
	stats = cache.Stats()

	expectedUtilization := float64(20) / float64(maxSize) * 100
	if stats.UtilizationPct != expectedUtilization {
		t.Errorf("Expected utilization %.2f%%, got %.2f%%", expectedUtilization, stats.UtilizationPct)
	}

	// Test hit/miss counts
	cache.Get("clip1")   // hit
	cache.Get("missing") // miss

	stats = cache.Stats()
	if stats.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Errorf("Expected miss count 1, got %d", stats.MissCount)
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewNormalizedClipCache(10000, logging.NopLogger)

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			clipID := fmt.Sprintf("clip-%d", index)
			data := []byte(fmt.Sprintf("data for clip %d", index))

			// Set data
			cache.Set(clipID, data)

			// Get data multiple times
			for j := 0; j < 10; j++ {
				_, _ = cache.Get(clipID)
			}

			// Delete if even index
			if index%2 == 0 {
				cache.Delete(clipID)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no panic occurred and cache is in a valid state
	stats := cache.Stats()
	if stats.TotalSize < 0 {
		t.Error("Cache size should not be negative after concurrent operations")
	}
}
