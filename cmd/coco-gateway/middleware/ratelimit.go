// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/api"
)

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	mu       sync.RWMutex
	tokens   map[string]*bucket
	rate     float64
	capacity int64
	refill   time.Duration
}

type bucket struct {
	tokens     int64
	lastRefill time.Time
}

// NewRateLimiter creates a new RateLimiter with the given rate, capacity, and refill duration.
func NewRateLimiter(rate float64, capacity int64, refill time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:   make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
		refill:   refill,
	}
}

// Allow checks if a request with the given key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.tokens[key]

	if !exists {
		rl.tokens[key] = &bucket{
			tokens:     rl.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastRefill)
	refills := int64(elapsed / rl.refill)
	if refills > 0 {
		b.tokens += refills * rl.capacity
		if b.tokens > rl.capacity {
			b.tokens = rl.capacity
		}
		b.lastRefill = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// Remaining returns the number of tokens remaining for the given key.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	b, exists := rl.tokens[key]
	if !exists {
		return int(rl.capacity)
	}
	return int(b.tokens)
}

// ResetTime returns the Unix timestamp when the bucket will be refilled.
func (rl *RateLimiter) ResetTime(key string) int64 {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	b, exists := rl.tokens[key]
	if !exists {
		return time.Now().Add(rl.refill).Unix()
	}
	remainingRefills := (rl.capacity - b.tokens) / rl.capacity
	return b.lastRefill.Add(time.Duration(remainingRefills) * rl.refill).Unix()
}

// RateLimitResponseWriter wraps http.ResponseWriter to add rate limit headers.
type RateLimitResponseWriter struct {
	http.ResponseWriter
	limit     int
	remaining int
	reset     int64
}

// WriteHeader adds rate limit headers before delegating to the underlying ResponseWriter.
func (w *RateLimitResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", w.limit))
	w.ResponseWriter.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", w.remaining))
	w.ResponseWriter.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", w.reset))
	w.ResponseWriter.WriteHeader(code)
}

// RateLimit returns a middleware that rate limits requests using the given RateLimiter.
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.RemoteAddr
			}

			allowed := limiter.Allow(key)
			remaining := limiter.Remaining(key)
			reset := limiter.ResetTime(key)

			rw := &RateLimitResponseWriter{
				ResponseWriter: w,
				limit:          100,
				remaining:      remaining,
				reset:          reset,
			}

			if !allowed {
				http.Error(rw, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(rw, r)
		})
	}
}

// FixedWindowLimiter implements a fixed window rate limiter.
type FixedWindowLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewFixedWindowLimiter creates a new FixedWindowLimiter with the given limit and window.
func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request with the given key is allowed under the fixed window.
func (f *FixedWindowLimiter) Allow(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-f.window)

	requests := f.requests[key]
	var valid []time.Time

	for _, t := range requests {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= f.limit {
		f.requests[key] = valid
		return false
	}

	f.requests[key] = append(valid, now)
	return true
}
