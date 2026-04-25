// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// Load Balancer - Multiple Algorithms
// =============================================================================

type Backend struct {
	URL     string
	Weight  int
	Active  bool
	Healthy bool
	// Stats
	Requests int64
	Failures int64
	LastFail time.Time
	Latency  time.Duration
}

type LoadBalancer interface {
	// Next returns the next backend to use
	Next() *Backend
	// MarkHealthy marks a backend as healthy
	MarkHealthy(url string)
	// MarkUnhealthy marks a backend as unhealthy
	MarkUnhealthy(url string)
	// Stats returns current stats
	Stats() map[string]interface{}
}

// =============================================================================
// Round Robin Load Balancer
// =============================================================================

type RoundRobinLB struct {
	mu       sync.RWMutex
	backends []*Backend
	current  int
}

func NewRoundRobinLB(backends []string) *RoundRobinLB {
	var be []*Backend
	for _, url := range backends {
		be = append(be, &Backend{
			URL:     url,
			Weight:  1,
			Active:  true,
			Healthy: true,
		})
	}
	return &RoundRobinLB{
		backends: be,
		current:  0,
	}
}

func (lb *RoundRobinLB) Next() *Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Try to find next active backend
	for i := 0; i < len(lb.backends); i++ {
		idx := (lb.current + i) % len(lb.backends)
		be := lb.backends[idx]

		if be.Active && be.Healthy {
			lb.current = (idx + 1) % len(lb.backends)
			be.Requests++
			return be
		}
	}

	// No healthy backend found
	return nil
}

func (lb *RoundRobinLB) MarkHealthy(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, be := range lb.backends {
		if be.URL == url {
			be.Active = true
			be.Healthy = true
			log.Printf("[lb] Backend %s marked healthy", url)
			return
		}
	}
}

func (lb *RoundRobinLB) MarkUnhealthy(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, be := range lb.backends {
		if be.URL == url {
			be.Healthy = false
			log.Printf("[lb] Backend %s marked unhealthy", url)
			return
		}
	}
}

func (lb *RoundRobinLB) Stats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var stats []map[string]interface{}
	for _, be := range lb.backends {
		stats = append(stats, map[string]interface{}{
			"url":       be.URL,
			"active":    be.Active,
			"healthy":   be.Healthy,
			"requests":  be.Requests,
			"failures":  be.Failures,
			"last_fail": be.LastFail,
		})
	}

	return map[string]interface{}{
		"algorithm": "round_robin",
		"backends":  stats,
		"current":   lb.current,
	}
}

// =============================================================================
// Least Connections Load Balancer
// =============================================================================

type LeastConnectionsLB struct {
	mu       sync.RWMutex
	backends []*Backend
}

func NewLeastConnectionsLB(backends []string) *LeastConnectionsLB {
	var be []*Backend
	for _, url := range backends {
		be = append(be, &Backend{
			URL:     url,
			Weight:  1,
			Active:  true,
			Healthy: true,
		})
	}
	return &LeastConnectionsLB{backends: be}
}

func (lb *LeastConnectionsLB) Next() *Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var best *Backend = nil
	var minConn int64 = 1 << 62

	for _, be := range lb.backends {
		if be.Active && be.Healthy {
			conns := be.Requests - be.Failures
			if conns < minConn {
				minConn = conns
				best = be
			}
		}
	}

	if best != nil {
		best.Requests++
	}

	return best
}

func (lb *LeastConnectionsLB) MarkHealthy(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, be := range lb.backends {
		if be.URL == url {
			be.Active = true
			be.Healthy = true
			return
		}
	}
}

func (lb *LeastConnectionsLB) MarkUnhealthy(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, be := range lb.backends {
		if be.URL == url {
			be.Healthy = false
			return
		}
	}
}

func (lb *LeastConnectionsLB) Stats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var stats []map[string]interface{}
	for _, be := range lb.backends {
		stats = append(stats, map[string]interface{}{
			"url":      be.URL,
			"active":   be.Active,
			"healthy":  be.Healthy,
			"requests": be.Requests,
			"failures": be.Failures,
		})
	}

	return map[string]interface{}{
		"algorithm": "least_connections",
		"backends":  stats,
	}
}

// =============================================================================
// Weighted Load Balancer
// =============================================================================

type WeightedLB struct {
	mu       sync.RWMutex
	backends []*Backend
	current  int
}

func NewWeightedLB(backends map[string]int) *WeightedLB {
	var be []*Backend
	for url, weight := range backends {
		be = append(be, &Backend{
			URL:     url,
			Weight:  weight,
			Active:  true,
			Healthy: true,
		})
	}
	return &WeightedLB{backends: be, current: 0}
}

func (lb *WeightedLB) Next() *Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Weighted round-robin
	totalWeight := 0
	for _, be := range lb.backends {
		if be.Active && be.Healthy {
			totalWeight += be.Weight
		}
	}

	if totalWeight == 0 {
		return nil
	}

	r := rand.Intn(totalWeight)
	running := 0
	for _, be := range lb.backends {
		if be.Active && be.Healthy {
			running += be.Weight
			if r < running {
				be.Requests++
				return be
			}
		}
	}

	return nil
}

func (lb *WeightedLB) MarkHealthy(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, be := range lb.backends {
		if be.URL == url {
			be.Active = true
			be.Healthy = true
			return
		}
	}
}

func (lb *WeightedLB) MarkUnhealthy(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, be := range lb.backends {
		if be.URL == url {
			be.Healthy = false
			return
		}
	}
}

func (lb *WeightedLB) Stats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var stats []map[string]interface{}
	for _, be := range lb.backends {
		stats = append(stats, map[string]interface{}{
			"url":     be.URL,
			"weight":  be.Weight,
			"active":  be.Active,
			"healthy": be.Healthy,
		})
	}

	return map[string]interface{}{
		"algorithm": "weighted",
		"backends":  stats,
	}
}

// =============================================================================
// Load Balancer HTTP Handler
// =============================================================================

type LBHandler struct {
	lb      LoadBalancer
	backend string // single backend fallback
}

func NewLBHandler(lb LoadBalancer) *LBHandler {
	return &LBHandler{lb: lb}
}

func (h *LBHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	be := h.lb.Next()
	if be == nil {
		http.Error(w, "No healthy backends", http.StatusServiceUnavailable)
		return
	}

	// Proxy to backend
	url := be.URL + r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		req.Header[k] = v
	}
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)
	req.Header.Set("X-Forwarded-Host", r.Host)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.lb.MarkUnhealthy(be.URL)
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Mark healthy on success
	h.lb.MarkHealthy(be.URL)

	// Copy response
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Copy body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
