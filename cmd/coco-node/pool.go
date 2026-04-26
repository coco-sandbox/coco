package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"coco/pkg/pool"
	"coco/pkg/visor"
)

type NodePool struct {
	vmPool    *pool.Pool
	visorPool *visor.Pool
	mu        sync.RWMutex
	stats     PoolStats
}

type PoolStats struct {
	ActiveCount  int
	FreeCount    int
	BootedCount  int
	FailedCount  int
	LastRefillAt time.Time
}

func NewNodePool(visorPool *visor.Pool, cfg pool.PoolConfig) *NodePool {
	return &NodePool{
		vmPool:    pool.NewPool(visorPool, cfg),
		visorPool: visorPool,
	}
}

func (np *NodePool) Start(ctx context.Context) error {
	active, free := np.vmPool.Stats()
	log.Printf("Starting node pool (active: %d, free: %d)", active, free)
	return nil
}

func (np *NodePool) Acquire(ctx context.Context, sandboxID, template string) (*pool.PooledVM, error) {
	vm, err := np.vmPool.Acquire(ctx, sandboxID, template)
	if err != nil {
		np.mu.Lock()
		np.stats.FailedCount++
		np.mu.Unlock()
		return nil, fmt.Errorf("failed to acquire VM: %w", err)
	}

	np.mu.Lock()
	np.stats.BootedCount++
	np.mu.Unlock()

	return vm, nil
}

func (np *NodePool) Release(sandboxID string) error {
	err := np.vmPool.Release(sandboxID)
	if err != nil {
		np.mu.Lock()
		np.stats.FailedCount++
		np.mu.Unlock()
		return fmt.Errorf("failed to release VM: %w", err)
	}

	return nil
}

func (np *NodePool) GetStats() PoolStats {
	np.mu.RLock()
	defer np.mu.RUnlock()

	active, free := np.vmPool.Stats()
	return PoolStats{
		ActiveCount: active,
		FreeCount:   free,
	}
}

func (np *NodePool) Stop() error {
	log.Println("Stopping node pool")
	return np.vmPool.Stop()
}
