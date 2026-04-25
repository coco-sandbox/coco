// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// Rate Limiting Middleware - Token Bucket Algorithm
// =============================================================================

type RateLimitConfig struct {
	// Capacity is the maximum tokens in the bucket
	Capacity int
	// Rate is tokens per second
	Rate float64
	// Burst is the maximum burst allowed
	Burst int
	// ByTenant if true, rate limits by tenant ID; otherwise by IP
	ByTenant bool
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	capacity   float64
	rate       float64
	mu         sync.Mutex
}

type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*tokenBucket
	config   RateLimitConfig
	throttle map[string]time.Time
}

func newRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.Capacity == 0 {
		cfg.Capacity = 600
	}
	if cfg.Rate == 0 {
		cfg.Rate = 10 // 10 tokens per second
	}
	if cfg.Burst == 0 {
		cfg.Burst = 100
	}

	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		config:   cfg,
		throttle: make(map[string]time.Time),
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[key]

	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(rl.config.Capacity),
			lastRefill: now,
			capacity:   float64(rl.config.Capacity),
			rate:       rl.config.Rate,
		}
		rl.buckets[key] = bucket
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min_float64(bucket.capacity, bucket.tokens+elapsed*bucket.rate)
	bucket.lastRefill = now

	// Check if we can consume a token
	if bucket.tokens >= 1 {
		bucket.tokens -= 1
		return true
	}

	return false
}

func (rl *RateLimiter) Throttle(key string, duration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.throttle[key] = time.Now().Add(duration)
}

func (rl *RateLimiter) IsThrottled(key string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if t, ok := rl.throttle[key]; ok && time.Now().Before(t) {
		return true
	}
	return false
}

func (rl *RateLimiter) GetTokens(key string) float64 {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if bucket, ok := rl.buckets[key]; ok {
		return bucket.tokens
	}
	return 0
}

// RateLimitMiddleware creates a middleware that enforces rate limits
func RateLimitMiddleware(limiter *RateLimiter, byTenant bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string

			if byTenant {
				tenant := r.Header.Get("X-Tenant-ID")
				if tenant == "" {
					tenant = "default"
				}
				key = "tenant:" + tenant
			} else {
				key = "ip:" + remoteAddr(r)
			}

			// Check if throttled
			if limiter.IsThrottled(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests"}}`))
				return
			}

			// Check rate limit
			if !limiter.Allow(key) {
				limiter.Throttle(key, 60*time.Second)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests"}}`))
				return
			}

			// Add rate limit headers
			tokens := limiter.GetTokens(key)
			w.Header().Set("X-RateLimit-Limit", "600")
			w.Header().Set("X-RateLimit-Remaining", formatInt(int(tokens)))
			w.Header().Set("X-RateLimit-Reset", formatInt(int(time.Now().Add(time.Minute).Unix())))

			next.ServeHTTP(w, r)
		})
	}
}

func remoteAddr(r *http.Request) string {
	// Check X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to remote address
	return r.RemoteAddr
}

func min_float64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func formatInt(n int) string {
	return string([]byte{
		byte('0' + (n/100)%10),
		byte('0' + (n/10)%10),
		byte('0' + n%10),
	})
}

// =============================================================================
// Sliding Window Rate Limiter
// =============================================================================

type SlidingWindowLimiter struct {
	mu       sync.RWMutex
	windows  map[string][]time.Time
	maxReqs  int
	windowSec int
}

func NewSlidingWindowLimiter(maxReqs int, windowSec int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windows:   make(map[string][]time.Time),
		maxReqs:   maxReqs,
		windowSec: windowSec,
	}
}

func (sw *SlidingWindowLimiter) Allow(key string) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(sw.windowSec) * time.Second)

	// Filter out old entries
	var valid []time.Time
	for _, t := range sw.windows[key] {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= sw.maxReqs {
		sw.windows[key] = valid
		return false
	}

	valid = append(valid, now)
	sw.windows[key] = valid
	return true
}

func (sw *SlidingWindowLimiter) Remaining(key string) int {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(sw.windowSec) * time.Second)

	count := 0
	for _, t := range sw.windows[key] {
		if t.After(windowStart) {
			count++
		}
	}

	return sw.maxReqs - count
}
