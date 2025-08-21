package streaming

import (
	"container/list"
	"sync"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

// CacheEntry represents a cached normalized clip with metadata
type CacheEntry struct {
	ClipID       string
	Data         []byte
	AccessTime   time.Time
	CreationTime time.Time
	element      *list.Element // for LRU tracking
}

// NormalizedClipCache provides caching for normalized video clips with LRU eviction
type NormalizedClipCache interface {
	// Get retrieves a normalized clip from cache
	Get(clipID string) ([]byte, bool)
	// Set stores a normalized clip in cache
	Set(clipID string, data []byte)
	// Delete removes a clip from cache
	Delete(clipID string)
	// Clear removes all entries from cache
	Clear()
	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheStats provides information about cache performance and usage
type CacheStats struct {
	TotalSize      int64   // Total cache size in bytes
	MaxSize        int64   // Maximum cache size in bytes
	EntryCount     int     // Number of entries in cache
	HitCount       int64   // Number of cache hits
	MissCount      int64   // Number of cache misses
	EvictionCount  int64   // Number of entries evicted
	UtilizationPct float64 // Cache utilization percentage
}

// lruCache implements NormalizedClipCache with LRU eviction policy
type lruCache struct {
	mutex       sync.RWMutex
	maxSize     int64 // Maximum cache size in bytes
	currentSize int64 // Current cache size in bytes
	entries     map[string]*CacheEntry
	lruList     *list.List // Most recently used at front, least recently used at back
	logger      logging.Logger

	// Statistics
	hitCount      int64
	missCount     int64
	evictionCount int64
}

// NewNormalizedClipCache creates a new LRU cache with the specified maximum size in bytes
func NewNormalizedClipCache(maxSizeBytes int64, logger logging.Logger) NormalizedClipCache {
	if logger == nil {
		logger = logging.NopLogger
	}

	if maxSizeBytes <= 0 {
		logger.Warn("Invalid cache size provided, using default 100MB", "providedSize", maxSizeBytes)
		maxSizeBytes = 100 * 1024 * 1024 // 100MB default
	}

	cache := &lruCache{
		maxSize: maxSizeBytes,
		entries: make(map[string]*CacheEntry),
		lruList: list.New(),
		logger:  logger,
	}

	logger.Info("Initialized normalized clip cache", "maxSizeBytes", maxSizeBytes, "maxSizeMB", maxSizeBytes/(1024*1024))
	return cache
}

// Get retrieves a normalized clip from cache
func (c *lruCache) Get(clipID string) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.entries[clipID]
	if !exists {
		c.missCount++
		return nil, false
	}

	// Update access time and move to front (most recently used)
	entry.AccessTime = time.Now()
	c.lruList.MoveToFront(entry.element)

	c.hitCount++
	c.logger.Debug("Cache hit for clip", "clipID", clipID, "dataSize", len(entry.Data))

	// Return a copy of the data to prevent external modification
	dataCopy := make([]byte, len(entry.Data))
	copy(dataCopy, entry.Data)
	return dataCopy, true
}

// Set stores a normalized clip in cache
func (c *lruCache) Set(clipID string, data []byte) {
	if len(data) == 0 {
		c.logger.Warn("Attempted to cache empty data", "clipID", clipID)
		return
	}

	// Don't cache if the data is larger than the entire cache
	if int64(len(data)) > c.maxSize {
		c.logger.Warn("Data too large for cache", "clipID", clipID, "dataSize", len(data), "maxSize", c.maxSize)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Check if entry already exists
	if existingEntry, exists := c.entries[clipID]; exists {
		// Update existing entry
		oldSize := int64(len(existingEntry.Data))
		existingEntry.Data = make([]byte, len(data))
		copy(existingEntry.Data, data)
		existingEntry.AccessTime = now
		c.currentSize = c.currentSize - oldSize + int64(len(data))
		c.lruList.MoveToFront(existingEntry.element)
		c.logger.Debug("Updated existing cache entry", "clipID", clipID, "oldSize", oldSize, "newSize", len(data))
	} else {
		// Create new entry
		entry := &CacheEntry{
			ClipID:       clipID,
			Data:         make([]byte, len(data)),
			AccessTime:   now,
			CreationTime: now,
		}
		copy(entry.Data, data)

		// Add to front of LRU list
		entry.element = c.lruList.PushFront(entry)
		c.entries[clipID] = entry
		c.currentSize += int64(len(data))

		c.logger.Debug("Added new cache entry", "clipID", clipID, "dataSize", len(data))
	}

	// Evict entries if necessary to stay within size limit
	c.evictIfNecessary()

	c.logger.Debug("Cache state after set", "currentSize", c.currentSize, "maxSize", c.maxSize, "entryCount", len(c.entries))
}

// Delete removes a clip from cache
func (c *lruCache) Delete(clipID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, exists := c.entries[clipID]; exists {
		c.removeEntry(entry)
		c.logger.Debug("Deleted cache entry", "clipID", clipID)
	}
}

// Clear removes all entries from cache
func (c *lruCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entryCount := len(c.entries)
	c.entries = make(map[string]*CacheEntry)
	c.lruList = list.New()
	c.currentSize = 0

	c.logger.Info("Cleared cache", "removedEntries", entryCount)
}

// Stats returns cache statistics
func (c *lruCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	utilizationPct := 0.0
	if c.maxSize > 0 {
		utilizationPct = float64(c.currentSize) / float64(c.maxSize) * 100
	}

	return CacheStats{
		TotalSize:      c.currentSize,
		MaxSize:        c.maxSize,
		EntryCount:     len(c.entries),
		HitCount:       c.hitCount,
		MissCount:      c.missCount,
		EvictionCount:  c.evictionCount,
		UtilizationPct: utilizationPct,
	}
}

// evictIfNecessary removes least recently used entries until cache is within size limit
func (c *lruCache) evictIfNecessary() {
	for c.currentSize > c.maxSize && c.lruList.Len() > 0 {
		// Remove from back (least recently used)
		element := c.lruList.Back()
		if element != nil {
			entry := element.Value.(*CacheEntry)
			c.removeEntry(entry)
			c.evictionCount++
			c.logger.Debug("Evicted cache entry", "clipID", entry.ClipID, "dataSize", len(entry.Data), "age", time.Since(entry.CreationTime))
		}
	}
}

// removeEntry removes an entry from both the map and LRU list
func (c *lruCache) removeEntry(entry *CacheEntry) {
	delete(c.entries, entry.ClipID)
	c.lruList.Remove(entry.element)
	c.currentSize -= int64(len(entry.Data))
}
