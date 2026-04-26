package rate

import (
	"sync"
	"time"
)

type Bucket struct {
	mu       sync.Mutex
	rate     float64
	capacity float64
	tokens   float64
	lastTime time.Time
}

func NewBucket(rate float64, capacity float64) *Bucket {
	return &Bucket{
		rate:     rate,
		capacity: capacity,
		tokens:   capacity,
		lastTime: time.Now(),
	}
}

func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.lastTime = now

	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

func (b *Bucket) AllowN(n int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.lastTime = now

	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}

	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true
	}

	return false
}

func (b *Bucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()

	tokens := b.tokens + elapsed*b.rate
	if tokens > b.capacity {
		return b.capacity
	}

	return tokens
}

func (b *Bucket) SetRate(rate float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rate = rate
}

func (b *Bucket) SetCapacity(capacity float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.capacity = capacity
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
}

func (b *Bucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens = b.capacity
	b.lastTime = time.Now()
}

type MultiBucket struct {
	mu      sync.RWMutex
	buckets map[string]*Bucket
	defaultRate     float64
	defaultCapacity float64
}

func NewMultiBucket(defaultRate, defaultCapacity float64) *MultiBucket {
	return &MultiBucket{
		buckets:         make(map[string]*Bucket),
		defaultRate:     defaultRate,
		defaultCapacity: defaultCapacity,
	}
}

func (m *MultiBucket) Allow(key string) bool {
	bucket := m.getBucket(key)
	return bucket.Allow()
}

func (m *MultiBucket) AllowN(key string, n int) bool {
	bucket := m.getBucket(key)
	return bucket.AllowN(n)
}

func (m *MultiBucket) getBucket(key string) *Bucket {
	m.mu.RLock()
	bucket, ok := m.buckets[key]
	m.mu.RUnlock()

	if ok {
		return bucket
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if bucket, ok = m.buckets[key]; ok {
		return bucket
	}

	bucket = NewBucket(m.defaultRate, m.defaultCapacity)
	m.buckets[key] = bucket

	return bucket
}

func (m *MultiBucket) Remove(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.buckets, key)
}

func (m *MultiBucket) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets = make(map[string]*Bucket)
}

func (m *MultiBucket) SetRate(key string, rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bucket, ok := m.buckets[key]; ok {
		bucket.SetRate(rate)
	}
}

func (m *MultiBucket) SetCapacity(key string, capacity float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bucket, ok := m.buckets[key]; ok {
		bucket.SetCapacity(capacity)
	}
}
