package jira

import (
	"sync"
	"time"
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
	ttl     time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]CacheEntry)
}
