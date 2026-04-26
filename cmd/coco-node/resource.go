package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type ResourceTracker struct {
	mu         sync.RWMutex
	nodeID     string
	totalMemMB uint64
	usedMemMB  uint64
	availMemMB uint64
	totalCPU   int
	usedCPU    int
	availCPU   int
	sandboxMem map[string]uint64
	sandboxCPU map[string]int
	updatedAt  time.Time
}

func NewResourceTracker(nodeID string, totalMemMB uint64, totalCPU int) *ResourceTracker {
	return &ResourceTracker{
		nodeID:     nodeID,
		totalMemMB: totalMemMB,
		totalCPU:   totalCPU,
		availMemMB: totalMemMB,
		availCPU:   totalCPU,
		sandboxMem: make(map[string]uint64),
		sandboxCPU: make(map[string]int),
		updatedAt:  time.Now(),
	}
}

func (rt *ResourceTracker) Allocate(sandboxID string, memMB uint64, vCPUs int) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if memMB > rt.availMemMB {
		return fmt.Errorf("insufficient memory: requested %dMB, available %dMB", memMB, rt.availMemMB)
	}

	if vCPUs > rt.availCPU {
		return fmt.Errorf("insufficient CPU: requested %d, available %d", vCPUs, rt.availCPU)
	}

	rt.sandboxMem[sandboxID] = memMB
	rt.sandboxCPU[sandboxID] = vCPUs

	rt.usedMemMB += memMB
	rt.availMemMB -= memMB
	rt.usedCPU += vCPUs
	rt.availCPU -= vCPUs
	rt.updatedAt = time.Now()

	return nil
}

func (rt *ResourceTracker) Release(sandboxID string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	memMB, ok := rt.sandboxMem[sandboxID]
	if !ok {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}

	vCPUs := rt.sandboxCPU[sandboxID]

	delete(rt.sandboxMem, sandboxID)
	delete(rt.sandboxCPU, sandboxID)

	rt.usedMemMB -= memMB
	rt.availMemMB += memMB
	rt.usedCPU -= vCPUs
	rt.availCPU += vCPUs
	rt.updatedAt = time.Now()

	return nil
}

func (rt *ResourceTracker) GetUsage() ResourceUsage {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ResourceUsage{
		NodeID:         rt.nodeID,
		TotalMemMB:     rt.totalMemMB,
		UsedMemMB:      rt.usedMemMB,
		AvailableMemMB: rt.availMemMB,
		TotalCPU:       rt.totalCPU,
		UsedCPU:        rt.usedCPU,
		AvailableCPU:   rt.availCPU,
		SandboxCount:   len(rt.sandboxMem),
		UpdatedAt:      rt.updatedAt,
		HostMemUsed:    memStats.Alloc / (1024 * 1024),
	}
}

func (rt *ResourceTracker) CanAllocate(memMB uint64, vCPUs int) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return memMB <= rt.availMemMB && vCPUs <= rt.availCPU
}

func (rt *ResourceTracker) GetSandboxResources(sandboxID string) (uint64, int, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	memMB, memOk := rt.sandboxMem[sandboxID]
	vCPUs, cpuOk := rt.sandboxCPU[sandboxID]

	if !memOk || !cpuOk {
		return 0, 0, false
	}

	return memMB, vCPUs, true
}

func (rt *ResourceTracker) Refresh() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.updatedAt = time.Now()
}

type ResourceUsage struct {
	NodeID         string
	TotalMemMB     uint64
	UsedMemMB      uint64
	AvailableMemMB uint64
	TotalCPU       int
	UsedCPU        int
	AvailableCPU   int
	SandboxCount   int
	UpdatedAt      time.Time
	HostMemUsed    uint64
}
