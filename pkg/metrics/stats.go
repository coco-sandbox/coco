package metrics

import (
	"sync"
	"time"
)

type Stats struct {
	mu           sync.RWMutex
	sandboxCount int
	nodeCount    int
	execCount    int
	errorsCount  int
	createdAt    time.Time
	lastUpdate   time.Time
}

var globalStats = &Stats{
	createdAt: time.Now(),
}

func GetStats() Stats {
	globalStats.mu.RLock()
	defer globalStats.mu.RUnlock()

	return Stats{
		sandboxCount: globalStats.sandboxCount,
		nodeCount:    globalStats.nodeCount,
		execCount:    globalStats.execCount,
		errorsCount:  globalStats.errorsCount,
		createdAt:    globalStats.createdAt,
		lastUpdate:   globalStats.lastUpdate,
	}
}

func IncrementSandboxCount() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.sandboxCount++
	globalStats.lastUpdate = time.Now()
}

func DecrementSandboxCount() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	if globalStats.sandboxCount > 0 {
		globalStats.sandboxCount--
	}
	globalStats.lastUpdate = time.Now()
}

func SetSandboxCount(count int) {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.sandboxCount = count
	globalStats.lastUpdate = time.Now()
}

func IncrementNodeCount() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.nodeCount++
	globalStats.lastUpdate = time.Now()
}

func DecrementNodeCount() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	if globalStats.nodeCount > 0 {
		globalStats.nodeCount--
	}
	globalStats.lastUpdate = time.Now()
}

func SetNodeCount(count int) {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.nodeCount = count
	globalStats.lastUpdate = time.Now()
}

func IncrementExecCount() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.execCount++
	globalStats.lastUpdate = time.Now()
}

func IncrementErrorCount() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.errorsCount++
	globalStats.lastUpdate = time.Now()
}

func (s *Stats) SandboxCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sandboxCount
}

func (s *Stats) NodeCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nodeCount
}

func (s *Stats) ExecCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.execCount
}

func (s *Stats) ErrorCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errorsCount
}

func (s *Stats) Uptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.createdAt)
}

type Counter struct {
	mu    sync.Mutex
	value uint64
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Add(n uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += n
}

func (c *Counter) Value() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
}

type Gauge struct {
	mu    sync.Mutex
	value float64
}

func (g *Gauge) Set(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = v
}

func (g *Gauge) Add(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value += v
}

func (g *Gauge) Sub(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value -= v
}

func (g *Gauge) Value() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.value
}
