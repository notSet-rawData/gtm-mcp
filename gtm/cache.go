package gtm

import (
	"sync"
	"time"
)

// readCache provides a TTL-based cache for read-only GTM API responses.
// It invalidates all entries for a workspace when a write operation occurs.
type readCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// defaultCacheTTL is the default time-to-live for cached read responses.
const defaultCacheTTL = 30 * time.Second

// globalCache is the package-level read cache instance.
var globalCache = newReadCache(defaultCacheTTL)

// newReadCache creates a new cache with the given TTL.
func newReadCache(ttl time.Duration) *readCache {
	return &readCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached value. Returns (value, true) on hit, (nil, false) on miss.
func (c *readCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

// Set stores a value in the cache with the configured TTL.
func (c *readCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// InvalidateWorkspace removes all cached entries for a given workspace.
// Called by write operations (create, update, delete) to ensure consistency.
func (c *readCache) InvalidateWorkspace(accountID, containerID, workspaceID string) {
	prefix := workspaceCachePrefix(accountID, containerID, workspaceID)
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
}

// InvalidateAll clears all cached entries.
func (c *readCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}

// workspaceCachePrefix returns the key prefix for a workspace's cached data.
func workspaceCachePrefix(accountID, containerID, workspaceID string) string {
	return accountID + "/" + containerID + "/" + workspaceID + "/"
}

// WorkspaceCacheKey builds a unique cache key for a workspace-scoped operation.
// Includes clientID for per-user isolation to prevent cross-user data leakage.
func WorkspaceCacheKey(accountID, containerID, workspaceID, operation string, clientID ...string) string {
	key := workspaceCachePrefix(accountID, containerID, workspaceID) + operation
	if len(clientID) > 0 && clientID[0] != "" {
		key = clientID[0] + ":" + key
	}
	return key
}
