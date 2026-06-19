package gtm

import (
	"sync"
	"time"
)

type readCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

const defaultCacheTTL = 30 * time.Second

var globalCache = newReadCache(defaultCacheTTL)

func newReadCache(ttl time.Duration) *readCache {
	return &readCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

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

func (c *readCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

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

func (c *readCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}

func workspaceCachePrefix(accountID, containerID, workspaceID string) string {
	return accountID + "/" + containerID + "/" + workspaceID + "/"
}

func WorkspaceCacheKey(accountID, containerID, workspaceID, operation string, clientID ...string) string {
	key := workspaceCachePrefix(accountID, containerID, workspaceID) + operation
	if len(clientID) > 0 && clientID[0] != "" {
		key = clientID[0] + ":" + key
	}
	return key
}
