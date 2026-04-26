package rate

import (
	"fmt"
	"sync"
	"time"
)

type Limiter struct {
	mu           sync.RWMutex
	limits       map[string]*Limit
	buckets      map[string]*MultiBucket
	defaultRPS   float64
	defaultBurst int
}

type Limit struct {
	SandboxID string
	RPS       float64
	Burst     int
	Direction string
	Proto     string
	Port      uint16
	Enabled   bool
	CreatedAt time.Time
}

func NewLimiter(defaultRPS float64, defaultBurst int) *Limiter {
	return &Limiter{
		limits:       make(map[string]*Limit),
		buckets:      make(map[string]*MultiBucket),
		defaultRPS:   defaultRPS,
		defaultBurst: defaultBurst,
	}
}

func (l *Limiter) SetLimit(limit *Limit) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := l.makeKey(limit.SandboxID, limit.Direction, limit.Proto, limit.Port)

	limit.CreatedAt = time.Now()
	limit.Enabled = true

	l.limits[key] = limit

	if _, ok := l.buckets[key]; !ok {
		l.buckets[key] = NewMultiBucket(limit.RPS, float64(limit.Burst))
	}

	return nil
}

func (l *Limiter) RemoveLimit(sandboxID, direction, proto string, port uint16) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := l.makeKey(sandboxID, direction, proto, port)

	delete(l.limits, key)
	delete(l.buckets, key)

	return nil
}

func (l *Limiter) Allow(sandboxID, direction, proto string, port uint16, tokens int) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	key := l.makeKey(sandboxID, direction, proto, port)

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = NewMultiBucket(l.defaultRPS, float64(l.defaultBurst))
	}

	if tokens <= 1 {
		return bucket.Allow(key)
	}

	return bucket.AllowN(key, tokens)
}

func (l *Limiter) GetLimit(sandboxID, direction, proto string, port uint16) (*Limit, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	key := l.makeKey(sandboxID, direction, proto, port)

	limit, ok := l.limits[key]
	return limit, ok
}

func (l *Limiter) ListLimits() []*Limit {
	l.mu.RLock()
	defer l.mu.RUnlock()

	limits := make([]*Limit, 0, len(l.limits))
	for _, limit := range l.limits {
		limits = append(limits, limit)
	}

	return limits
}

func (l *Limiter) UpdateLimit(sandboxID, direction, proto string, port uint16, rps float64, burst int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := l.makeKey(sandboxID, direction, proto, port)

	limit, ok := l.limits[key]
	if !ok {
		return fmt.Errorf("limit not found")
	}

	limit.RPS = rps
	limit.Burst = burst

	if bucket, ok := l.buckets[key]; ok {
		bucket.SetRate(key, rps)
		bucket.SetCapacity(key, float64(burst))
	}

	return nil
}

func (l *Limiter) makeKey(sandboxID, direction, proto string, port uint16) string {
	if port == 0 {
		return fmt.Sprintf("%s:%s:%s", sandboxID, direction, proto)
	}
	return fmt.Sprintf("%s:%s:%s:%d", sandboxID, direction, proto, port)
}

func (l *Limiter) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := make(map[string]interface{})

	for key, limit := range l.limits {
		stats[key] = map[string]interface{}{
			"rps":     limit.RPS,
			"burst":   limit.Burst,
			"enabled": limit.Enabled,
		}
	}

	return stats
}
