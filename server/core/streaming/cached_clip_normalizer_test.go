package streaming

import (
	"fmt"
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
)

// mockClipNormalizer for testing
type mockClipNormalizer struct {
	normalizeFunc func(clip *videos.DecryptedClip) ([]byte, error)
	callCount     int
}

func (m *mockClipNormalizer) NormalizeClip(clip *videos.DecryptedClip) ([]byte, error) {
	m.callCount++
	if m.normalizeFunc != nil {
		return m.normalizeFunc(clip)
	}
	return []byte(fmt.Sprintf("normalized-%s", clip.ID)), nil
}

func createTestDecryptedClip(id string) *videos.DecryptedClip {
	return &videos.DecryptedClip{
		ID:            id,
		ClientID:      "test-client",
		Title:         fmt.Sprintf("Test Clip %s", id),
		TimeStamp:     time.Now(),
		Duration:      time.Duration(30 * time.Second),
		HasMotion:     true,
		Video:         []byte("test video data"),
		VideoWidth:    1920,
		VideoHeight:   1080,
		VideoMimeType: "video/mp4",
	}
}

func TestNewCachedClipNormalizer(t *testing.T) {
	mockNormalizer := &mockClipNormalizer{}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)

	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	if cachedNormalizer == nil {
		t.Fatal("Expected cached normalizer to be created")
	}
	if cachedNormalizer.normalizer != mockNormalizer {
		t.Error("Expected normalizer to be set correctly")
	}
	if cachedNormalizer.cache != cache {
		t.Error("Expected cache to be set correctly")
	}
}

func TestCachedClipNormalizerCacheHit(t *testing.T) {
	mockNormalizer := &mockClipNormalizer{}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	clip := createTestDecryptedClip("test-clip-1")

	// First call should normalize and cache
	result1, err := cachedNormalizer.NormalizeClip(clip)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if mockNormalizer.callCount != 1 {
		t.Errorf("Expected normalizer to be called once, got %d calls", mockNormalizer.callCount)
	}

	// Second call should use cache
	result2, err := cachedNormalizer.NormalizeClip(clip)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if mockNormalizer.callCount != 1 {
		t.Errorf("Expected normalizer to still be called only once, got %d calls", mockNormalizer.callCount)
	}

	// Results should be identical
	if string(result1) != string(result2) {
		t.Errorf("Expected identical results, got '%s' and '%s'", string(result1), string(result2))
	}

	// Verify cache stats
	stats := cachedNormalizer.GetCacheStats()
	if stats.HitCount != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Errorf("Expected 1 cache miss, got %d", stats.MissCount)
	}
}

func TestCachedClipNormalizerDifferentClips(t *testing.T) {
	mockNormalizer := &mockClipNormalizer{}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	clip1 := createTestDecryptedClip("test-clip-1")
	clip2 := createTestDecryptedClip("test-clip-2")

	// Normalize both clips
	result1, err := cachedNormalizer.NormalizeClip(clip1)
	if err != nil {
		t.Fatalf("Expected no error for clip1, got %v", err)
	}

	result2, err := cachedNormalizer.NormalizeClip(clip2)
	if err != nil {
		t.Fatalf("Expected no error for clip2, got %v", err)
	}

	// Both should be normalized (cache miss for each)
	if mockNormalizer.callCount != 2 {
		t.Errorf("Expected normalizer to be called twice, got %d calls", mockNormalizer.callCount)
	}

	// Results should be different
	if string(result1) == string(result2) {
		t.Error("Expected different results for different clips")
	}

	// Both clips should now be cached
	result1_cached, err := cachedNormalizer.NormalizeClip(clip1)
	if err != nil {
		t.Fatalf("Expected no error for cached clip1, got %v", err)
	}
	result2_cached, err := cachedNormalizer.NormalizeClip(clip2)
	if err != nil {
		t.Fatalf("Expected no error for cached clip2, got %v", err)
	}

	// Should still be called only twice (cache hits)
	if mockNormalizer.callCount != 2 {
		t.Errorf("Expected normalizer to still be called only twice, got %d calls", mockNormalizer.callCount)
	}

	// Cached results should match original results
	if string(result1) != string(result1_cached) {
		t.Error("Cached result for clip1 doesn't match original")
	}
	if string(result2) != string(result2_cached) {
		t.Error("Cached result for clip2 doesn't match original")
	}
}

func TestCachedClipNormalizerNormalizationError(t *testing.T) {
	expectedError := fmt.Errorf("normalization failed")
	mockNormalizer := &mockClipNormalizer{
		normalizeFunc: func(clip *videos.DecryptedClip) ([]byte, error) {
			return nil, expectedError
		},
	}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	clip := createTestDecryptedClip("test-clip-1")

	_, err := cachedNormalizer.NormalizeClip(clip)
	if err == nil {
		t.Fatal("Expected error from normalization")
	}
	if err != expectedError {
		t.Errorf("Expected specific error, got %v", err)
	}

	// Error should not be cached
	stats := cachedNormalizer.GetCacheStats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected no cache entries after error, got %d", stats.EntryCount)
	}
}

func TestCachedClipNormalizerNilClip(t *testing.T) {
	mockNormalizer := &mockClipNormalizer{}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	_, err := cachedNormalizer.NormalizeClip(nil)
	if err == nil {
		t.Fatal("Expected error for nil clip")
	}
	if mockNormalizer.callCount != 0 {
		t.Errorf("Expected normalizer not to be called for nil clip, got %d calls", mockNormalizer.callCount)
	}
}

func TestCachedClipNormalizerClearCache(t *testing.T) {
	mockNormalizer := &mockClipNormalizer{}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	clip := createTestDecryptedClip("test-clip-1")

	// Normalize and cache
	_, err := cachedNormalizer.NormalizeClip(clip)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify it's cached
	stats := cachedNormalizer.GetCacheStats()
	if stats.EntryCount != 1 {
		t.Errorf("Expected 1 cache entry, got %d", stats.EntryCount)
	}

	// Clear cache
	cachedNormalizer.ClearCache()

	// Verify cache is empty
	stats = cachedNormalizer.GetCacheStats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", stats.EntryCount)
	}
	if stats.TotalSize != 0 {
		t.Errorf("Expected 0 cache size after clear, got %d", stats.TotalSize)
	}

	// Next call should normalize again
	_, err = cachedNormalizer.NormalizeClip(clip)
	if err != nil {
		t.Fatalf("Expected no error after cache clear, got %v", err)
	}
	if mockNormalizer.callCount != 2 {
		t.Errorf("Expected normalizer to be called twice (before and after clear), got %d calls", mockNormalizer.callCount)
	}
}

func TestCachedClipNormalizerGetCacheStats(t *testing.T) {
	mockNormalizer := &mockClipNormalizer{}
	cache := NewNormalizedClipCache(1024, logging.NopLogger)
	cachedNormalizer := NewCachedClipNormalizer(mockNormalizer, cache, logging.NopLogger)

	clip1 := createTestDecryptedClip("test-clip-1")
	clip2 := createTestDecryptedClip("test-clip-2")

	// Initial stats
	stats := cachedNormalizer.GetCacheStats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected initial entry count 0, got %d", stats.EntryCount)
	}
	if stats.HitCount != 0 {
		t.Errorf("Expected initial hit count 0, got %d", stats.HitCount)
	}
	if stats.MissCount != 0 {
		t.Errorf("Expected initial miss count 0, got %d", stats.MissCount)
	}

	// Add clips
	cachedNormalizer.NormalizeClip(clip1)
	cachedNormalizer.NormalizeClip(clip2)
	cachedNormalizer.NormalizeClip(clip1) // cache hit

	stats = cachedNormalizer.GetCacheStats()
	if stats.EntryCount != 2 {
		t.Errorf("Expected entry count 2, got %d", stats.EntryCount)
	}
	if stats.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", stats.HitCount)
	}
	if stats.MissCount != 2 {
		t.Errorf("Expected miss count 2, got %d", stats.MissCount)
	}
}
