package policy

import (
	"fmt"
	"sync"
	"time"
)

type Cache struct {
	mu              sync.RWMutex
	entries         map[string]*CacheEntry
	maxSize         int
	ttl             time.Duration
	cleanupInterval time.Duration
}

type CacheEntry struct {
	Action      Action
	Direction   Direction
	Conn        *Connection
	CreatedAt   time.Time
	AccessCount int
}

func NewCache() *Cache {
	c := &Cache{
		entries:         make(map[string]*CacheEntry),
		maxSize:         10000,
		ttl:             5 * time.Minute,
		cleanupInterval: 1 * time.Minute,
	}

	go c.cleanupLoop()

	return c
}

func (c *Cache) Get(sandboxID string, direction Direction, conn *Connection) (Action, bool) {
	key := c.makeKey(sandboxID, direction, conn)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return 0, false
	}

	if time.Since(entry.CreatedAt) > c.ttl {
		return 0, false
	}

	entry.AccessCount++

	return entry.Action, true
}

func (c *Cache) Set(sandboxID string, direction Direction, conn *Connection, action Action) {
	key := c.makeKey(sandboxID, direction, conn)

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &CacheEntry{
		Action:    action,
		Direction: direction,
		Conn:      conn,
		CreatedAt: time.Now(),
	}
}

func (c *Cache) Invalidate(sandboxID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := sandboxID + ":"

	for key := range c.entries {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
}

func (c *Cache) Stats() (int, int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := len(c.entries)
	var size int64

	for _, entry := range c.entries {
		size += int64(len(fmt.Sprintf("%v", entry.Conn)))
	}

	return count, size
}

func (c *Cache) makeKey(sandboxID string, direction Direction, conn *Connection) string {
	return fmt.Sprintf("%s:%d:%s:%d:%d", sandboxID, direction, conn.SrcIP, conn.SrcPort, conn.DstPort)
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.ttl)

	for key, entry := range c.entries {
		if entry.CreatedAt.Before(cutoff) {
			delete(c.entries, key)
		}
	}
}
