package streaming

import (
	"fmt"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
)

// CachedClipNormalizer wraps a ClipNormalizer with caching functionality
type CachedClipNormalizer struct {
	normalizer ClipNormalizer
	cache      NormalizedClipCache
	logger     logging.Logger
}

// NewCachedClipNormalizer creates a new cached clip normalizer
func NewCachedClipNormalizer(normalizer ClipNormalizer, cache NormalizedClipCache, logger logging.Logger) *CachedClipNormalizer {
	if logger == nil {
		logger = logging.NopLogger
	}

	return &CachedClipNormalizer{
		normalizer: normalizer,
		cache:      cache,
		logger:     logger,
	}
}

// NormalizeClip normalizes a clip with caching support
func (c *CachedClipNormalizer) NormalizeClip(clip *videos.DecryptedClip) ([]byte, error) {
	if clip == nil {
		return nil, fmt.Errorf("clip cannot be nil")
	}

	// Try to get from cache first
	if cachedData, found := c.cache.Get(clip.ID); found {
		c.logger.Debug("Serving normalized clip from cache", "clipID", clip.ID, "dataSize", len(cachedData))
		return cachedData, nil
	}

	// Cache miss - normalize the clip
	c.logger.Debug("Cache miss, normalizing clip", "clipID", clip.ID)
	normalizedData, err := c.normalizer.NormalizeClip(clip)
	if err != nil {
		return nil, err
	}

	// Cache the normalized data
	c.cache.Set(clip.ID, normalizedData)
	c.logger.Debug("Cached normalized clip", "clipID", clip.ID, "dataSize", len(normalizedData))

	return normalizedData, nil
}

// GetCacheStats returns statistics about the cache
func (c *CachedClipNormalizer) GetCacheStats() CacheStats {
	return c.cache.Stats()
}

// ClearCache clears all cached entries
func (c *CachedClipNormalizer) ClearCache() {
	c.cache.Clear()
	c.logger.Info("Cleared normalized clip cache")
}
