package config

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/client"
)

const (
	// DefaultSettingsCacheTimeout is the default cache timeout period
	DefaultSettingsCacheTimeout = 5 * time.Minute
)

type ClientSettingsProvider struct {
	client          client.CaptureServerClient
	mutex           sync.RWMutex
	cachedSettings  *client.ClientSettingsResponse
	lastFetchTime   time.Time
	fetchInProgress bool
	cacheTimeout    time.Duration
}

// NewClientSettingsProvider creates a new ClientSettingsProvider
// It performs an initial fetch of settings and returns an error if this fails
// If cacheTimeout is 0, DefaultSettingsCacheTimeout is used
func NewClientSettingsProvider(client client.CaptureServerClient, cacheTimeout time.Duration) (*ClientSettingsProvider, error) {
	if cacheTimeout == 0 {
		cacheTimeout = DefaultSettingsCacheTimeout
	}

	provider := &ClientSettingsProvider{
		client:       client,
		cacheTimeout: cacheTimeout,
	}

	// Perform initial fetch to ensure we always have settings
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	settings, err := client.GetClientSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial client settings: %w", err)
	}

	provider.cachedSettings = settings
	provider.lastFetchTime = time.Now()

	return provider, nil
}

// GetSettings returns the current settings, implementing SettingsProvider interface
func (p *ClientSettingsProvider) GetSettings() client.ClientSettingsResponse {
	p.mutex.RLock()

	// Check if we need to refresh settings (we're guaranteed to have cachedSettings)
	needsRefresh := time.Since(p.lastFetchTime) > p.cacheTimeout
	fetchInProgress := p.fetchInProgress
	currentSettings := *p.cachedSettings // Safe to dereference since constructor ensures it's set

	p.mutex.RUnlock()

	// If settings are stale and no fetch is in progress, start async fetch
	if needsRefresh && !fetchInProgress {
		go p.fetchSettingsAsync()
	}

	// Return current settings (may be stale, but ensures no blocking)
	return currentSettings
}

// fetchSettingsAsync fetches settings in the background without blocking
func (p *ClientSettingsProvider) fetchSettingsAsync() {
	p.mutex.Lock()

	// Check if another goroutine already started fetching
	if p.fetchInProgress {
		p.mutex.Unlock()
		return
	}

	p.fetchInProgress = true
	p.mutex.Unlock()

	// Ensure we reset the fetchInProgress flag when done
	defer func() {
		p.mutex.Lock()
		p.fetchInProgress = false
		p.mutex.Unlock()
	}()

	// Fetch with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	settings, err := p.client.GetClientSettings(ctx)
	if err != nil {
		// Log error but don't update cache - keep using stale settings
		log.Printf("Failed to fetch client settings from server: %v", err)
		return
	}

	// Update cache with fresh settings
	p.mutex.Lock()
	p.cachedSettings = settings
	p.lastFetchTime = time.Now()
	p.mutex.Unlock()
}
