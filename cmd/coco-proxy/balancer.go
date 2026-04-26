package main

import (
	"math"
	"sync"
	"time"
)

type WeightedBalancer struct {
	backends []*Backend
	current  int
	mu       sync.Mutex
}

func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{}
}

func (wb *WeightedBalancer) AddBackend(backend *Backend) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	wb.backends = append(wb.backends, backend)
}

func (wb *WeightedBalancer) SelectBackend() *Backend {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if len(wb.backends) == 0 {
		return nil
	}

	totalWeight := 0
	for _, b := range wb.backends {
		totalWeight += b.Weight
	}

	r := time.Now().UnixNano() % int64(totalWeight)

	running := 0
	for _, b := range wb.backends {
		running += b.Weight
		if int64(running) > r {
			return b
		}
	}

	return wb.backends[0]
}

type LeastConnectionsBalancer struct {
	backends []*Backend
	mu        sync.Mutex
}

func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
	return &LeastConnectionsBalancer{}
}

func (lc *LeastConnectionsBalancer) AddBackend(backend *Backend) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.backends = append(lc.backends, backend)
}

func (lc *LeastConnectionsBalancer) SelectBackend() *Backend {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.backends) == 0 {
		return nil
	}

	minConns := math.MaxInt
	var selected *Backend

	for _, b := range lc.backends {
		if b.ActiveConns < minConns {
			minConns = b.ActiveConns
			selected = b
		}
	}

	return selected
}

func (lc *LeastConnectionsBalancer) RemoveBackend(backend *Backend) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i, b := range lc.backends {
		if b == backend {
			lc.backends = append(lc.backends[:i], lc.backends[i+1:]...)
			break
		}
	}
}
