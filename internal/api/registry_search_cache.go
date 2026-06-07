package api

import (
	"strings"
	"sync"
	"time"
)

const registrySearchCacheTTL = 45 * time.Second

type registrySearchCache struct {
	mu      sync.RWMutex
	entries map[string]registrySearchCacheEntry
}

type registrySearchCacheEntry struct {
	value     any
	expiresAt time.Time
}

func newRegistrySearchCache() *registrySearchCache {
	return &registrySearchCache{entries: map[string]registrySearchCacheEntry{}}
}

func (c *registrySearchCache) get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (c *registrySearchCache) set(key string, value any) {
	c.mu.Lock()
	c.entries[key] = registrySearchCacheEntry{value: value, expiresAt: time.Now().Add(registrySearchCacheTTL)}
	c.mu.Unlock()
}

func registrySearchCacheKey(parts ...string) string {
	for index, part := range parts {
		parts[index] = strings.TrimSpace(part)
	}
	return strings.Join(parts, "\x00")
}
