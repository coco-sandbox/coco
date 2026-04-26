package pool

import (
	"sync/atomic"
)

type PoolMetrics struct {
	activeCount   atomic.Int64
	idleCount     atomic.Int64
	acquiredCount atomic.Int64
	releasedCount atomic.Int64
	totalAcquired atomic.Int64
	totalReleased atomic.Int64
}

func NewPoolMetrics() *PoolMetrics {
	return &PoolMetrics{}
}

func (m *PoolMetrics) Active() int64 {
	return m.activeCount.Load()
}

func (m *PoolMetrics) Idle() int64 {
	return m.idleCount.Load()
}

func (m *PoolMetrics) Acquired() int64 {
	return m.acquiredCount.Load()
}

func (m *PoolMetrics) Released() int64 {
	return m.releasedCount.Load()
}

func (m *PoolMetrics) TotalAcquired() int64 {
	return m.totalAcquired.Load()
}

func (m *PoolMetrics) TotalReleased() int64 {
	return m.totalReleased.Load()
}

func (m *PoolMetrics) recordAcquire() {
	m.activeCount.Add(1)
	m.acquiredCount.Add(1)
	m.totalAcquired.Add(1)
}

func (m *PoolMetrics) recordRelease() {
	m.activeCount.Add(-1)
	m.releasedCount.Add(1)
	m.totalReleased.Add(1)
}

func (m *PoolMetrics) recordIdle() {
	m.idleCount.Add(1)
}

func (m *PoolMetrics) recordUse() {
	m.idleCount.Add(-1)
}
