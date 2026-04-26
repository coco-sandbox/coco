package middleware

import (
	"net/http"
	"sync"
	"time"
)

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

func NewRateLimiter(rate float64, capacity int64, refill time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:   make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
		refill:   refill,
	}
}

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

func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr

			if !limiter.Allow(key) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type FixedWindowLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

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
