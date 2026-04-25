// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// =============================================================================
// Config
// =============================================================================

const (
	ListenAddr    = ":4749"
	BackendAddr   = "localhost:4747"
	ReadTimeout   = 30 * time.Second
	WriteTimeout  = 30 * time.Second
	IdleTimeout   = 60 * time.Second
	ShutdownTO    = 10 * time.Second
)

// =============================================================================
// Main
// =============================================================================

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	log.Printf("cocogate starting on %s", ListenAddr)
	log.Printf("Backend: %s", BackendAddr)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create rate limiter
	rl := newRateLimiter(RateLimitConfig{
		Capacity: 600,
		Rate:     10,
		Burst:    100,
		ByTenant: true,
	})

	// Create load balancer
	lb := NewRoundRobinLB([]string{BackendAddr})

	// Create circuit breaker pool
	cbPool := NewCircuitBreakerPool(5, 30*time.Second)

	// Create proxy handler
	handler := newProxyHandlerV2(rl, lb, cbPool)

	// Create server
	server := &http.Server{
		Addr:         ListenAddr,
		Handler:      handler,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	// Start server
	go func() {
		log.Printf("Listening on %s", ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTO)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("cocogate stopped")
}

// =============================================================================
// Proxy Handler V2 (uses modular components)
// =============================================================================

type proxyHandlerV2 struct {
	rl *RateLimiter
	lb *RoundRobinLB
	cb *CircuitBreakerPool
}

func newProxyHandlerV2(rl *RateLimiter, lb *RoundRobinLB, cb *CircuitBreakerPool) *proxyHandlerV2 {
	return &proxyHandlerV2{
		rl: rl,
		lb: lb,
		cb: cb,
	}
}

func (h *proxyHandlerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get tenant for rate limiting
	tenant := r.Header.Get("X-Tenant-ID")
	if tenant == "" {
		tenant = "default"
	}

	// Check rate limit
	if !h.rl.Allow("tenant:" + tenant) {
		if h.rl.IsThrottled("tenant:" + tenant) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests"}}`))
			return
		}
	}

	// Get next backend
	be := h.lb.Next()
	if be == nil {
		http.Error(w, "No backends available", http.StatusServiceUnavailable)
		return
	}

	// Get circuit breaker for this backend
	cb := h.cb.GetBreaker(be.URL)
	if !cb.Allow() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":{"code":"CIRCUIT_OPEN","message":"Service temporarily unavailable"}}`))
		return
	}

	// Proxy to backend
	url := be.URL + r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		cb.RecordFailure()
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		req.Header[k] = v
	}
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		cb.RecordFailure()
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Record success
	cb.RecordSuccess()

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
