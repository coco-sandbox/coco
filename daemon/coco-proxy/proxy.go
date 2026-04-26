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
	mu           sync.RWMutex
	backends     map[string]*Backend
	balancer     LoadBalancer
	timeout      time.Duration
	maxRetries   int
	cache        *ResponseCache
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

type ResponseCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedResponse
	ttl   time.Duration
}

type CachedResponse struct {
	Data      []byte
	ExpiresAt time.Time
}

func NewProxy(timeout time.Duration, maxRetries int) *Proxy {
	return &Proxy{
		backends:   make(map[string]*Backend),
		balancer:   &RoundRobinBalancer{},
		timeout:    timeout,
		maxRetries: maxRetries,
		cache:      &ResponseCache{cache: make(map[string]*CachedResponse)},
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

func (c *ResponseCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if resp, ok := c.cache[key]; ok {
		if time.Now().Before(resp.ExpiresAt) {
			return resp.Data, true
		}
	}

	return nil, false
}

func (c *ResponseCache) Set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &CachedResponse{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *ResponseCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, resp := range c.cache {
		if now.After(resp.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}
