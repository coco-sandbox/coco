// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Proxy struct {
	mu         sync.RWMutex
	backends   map[string]*Backend
	balancer   LoadBalancer
	timeout    time.Duration
	maxRetries int
	cache      *Cache
}

type Backend struct {
	URL         *url.URL
	HealthCheck *http.Client
	ActiveConns int
	Weight      int
}

type LoadBalancer interface {
	SelectBackend() *Backend
}

type RoundRobinBalancer struct {
	backends []*Backend
	current  int
	mu       sync.Mutex
}

func NewProxy(timeout time.Duration, maxRetries int) *Proxy {
	return &Proxy{
		backends:   make(map[string]*Backend),
		balancer:   &RoundRobinBalancer{},
		timeout:    timeout,
		maxRetries: maxRetries,
		cache:      NewCache(1000, 5*time.Minute, 10*time.Minute),
	}
}

func (p *Proxy) AddBackend(name, addr string, weight int) error {
	backendURL, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("invalid backend address: %w", err)
	}

	backend := &Backend{
		URL:    backendURL,
		Weight: weight,
		HealthCheck: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	p.mu.Lock()
	p.backends[name] = backend
	p.mu.Unlock()

	log.Printf("Added backend %s -> %s (weight: %d)", name, addr, weight)
	return nil
}

func (p *Proxy) RemoveBackend(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.backends, name)
	log.Printf("Removed backend %s", name)
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := p.balancer.SelectBackend()
	if backend == nil {
		http.Error(w, "no backends available", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(backend.URL)
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	p.mu.Lock()
	backend.ActiveConns++
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		backend.ActiveConns--
		p.mu.Unlock()
	}()

	proxy.ServeHTTP(w, r)
}

func (rb *RoundRobinBalancer) SelectBackend() *Backend {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.backends) == 0 {
		return nil
	}

	backend := rb.backends[rb.current]
	rb.current = (rb.current + 1) % len(rb.backends)

	return backend
}