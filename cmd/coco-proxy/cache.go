package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
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
	Key         string
	Response    *CachedResponse
	CreatedAt   time.Time
	AccessCount int64
	Size        int64
}

type CachedResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func NewCache(maxSize int, ttl time.Duration, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		entries:         make(map[string]*CacheEntry),
		maxSize:         maxSize,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
	}

	go c.cleanupLoop()

	return c
}

func (c *Cache) Get(key string) (*CachedResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Since(entry.CreatedAt) > c.ttl {
		return nil, false
	}

	entry.AccessCount++
	return entry.Response, true
}

func (c *Cache) Set(key string, resp *CachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &CacheEntry{
		Key:       key,
		Response:  resp,
		CreatedAt: time.Now(),
		Size:      int64(len(resp.Body)),
	}

	c.entries[key] = entry

	c.evictIfNeeded()
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

func (c *Cache) evictIfNeeded() {
	totalSize := int64(0)
	for _, entry := range c.entries {
		totalSize += entry.Size
	}

	if int(totalSize) > c.maxSize {
		c.evictOldest()
	}
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

	now := time.Now()
	for key, entry := range c.entries {
		if now.Sub(entry.CreatedAt) > c.ttl {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) Stats() (int, int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := len(c.entries)
	var size int64
	for _, entry := range c.entries {
		size += entry.Size
	}

	return count, size
}

func HashRequest(r *http.Request) string {
	h := sha256.New()

	h.Write([]byte(r.Method))
	h.Write([]byte(r.URL.Path))
	h.Write([]byte(r.URL.RawQuery))

	for _, cookie := range r.Cookies() {
		h.Write([]byte(cookie.Name))
		h.Write([]byte(cookie.Value))
	}

	io.WriteString(h, r.Header.Get("Authorization"))

	return fmt.Sprintf("%x", h.Sum(nil))
}
