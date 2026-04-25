// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// Circuit Breaker - State Machine Pattern
// =============================================================================

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu             sync.RWMutex
	name           string
	state          CircuitState
	failureLimit   int
	failureCount   int
	successLimit   int
	successCount   int
	lastFailure    time.Time
	resetTimeout   time.Duration
	halfOpenMaxReq int
	halfOpenReqs   int
}

type CircuitBreakerConfig struct {
	Name             string
	FailureLimit     int
	SuccessLimit     int // Number of successes needed to close circuit
	ResetTimeout     time.Duration
	HalfOpenMaxReqs  int // Max requests in half-open state
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureLimit == 0 {
		cfg.FailureLimit = 5
	}
	if cfg.SuccessLimit == 0 {
		cfg.SuccessLimit = 3
	}
	if cfg.ResetTimeout == 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxReqs == 0 {
		cfg.HalfOpenMaxReqs = 3
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}

	return &CircuitBreaker{
		name:           cfg.Name,
		state:          CircuitClosed,
		failureLimit:   cfg.FailureLimit,
		successLimit:   cfg.SuccessLimit,
		resetTimeout:   cfg.ResetTimeout,
		halfOpenMaxReq: cfg.HalfOpenMaxReqs,
	}
}

// Allow checks if a request is allowed through the circuit breaker
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenReqs = 0
			log.Printf("[circuit] %s: Open -> Half-Open", cb.name)
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenReqs < cb.halfOpenMaxReq {
			cb.halfOpenReqs++
			return true
		}
		return false

	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0

	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successLimit {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.successCount = 0
			log.Printf("[circuit] %s: Half-Open -> Closed (success)", cb.name)
		}

	case CircuitOpen:
		// Shouldn't happen, but handle gracefully
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		cb.lastFailure = time.Now()
		if cb.failureCount >= cb.failureLimit {
			cb.state = CircuitOpen
			log.Printf("[circuit] %s: Closed -> Open (failures=%d)", cb.name, cb.failureCount)
		}

	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.halfOpenReqs = 0
		cb.lastFailure = time.Now()
		log.Printf("[circuit] %s: Half-Open -> Open (failure)", cb.name)

	case CircuitOpen:
		// Already open, just update last failure time
		cb.lastFailure = time.Now()
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":           cb.name,
		"state":          cb.state.String(),
		"failure_count":  cb.failureCount,
		"failure_limit":  cb.failureLimit,
		"success_count":  cb.successCount,
		"last_failure":   cb.lastFailure,
		"reset_timeout":  cb.resetTimeout,
	}
}

// =============================================================================
// Per-Backend Circuit Breakers
// =============================================================================

type CircuitBreakerPool struct {
	mu          sync.RWMutex
	breakers    map[string]*CircuitBreaker
	globalLimit int
	resetTO     time.Duration
}

func NewCircuitBreakerPool(globalLimit int, resetTO time.Duration) *CircuitBreakerPool {
	if globalLimit == 0 {
		globalLimit = 5
	}
	if resetTO == 0 {
		resetTO = 30 * time.Second
	}

	return &CircuitBreakerPool{
		breakers:    make(map[string]*CircuitBreaker),
		globalLimit: globalLimit,
		resetTO:     resetTO,
	}
}

func (pool *CircuitBreakerPool) GetBreaker(backend string) *CircuitBreaker {
	pool.mu.RLock()
	if cb, ok := pool.breakers[backend]; ok {
		pool.mu.RUnlock()
		return cb
	}
	pool.mu.RUnlock()

	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := pool.breakers[backend]; ok {
		return cb
	}

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         backend,
		FailureLimit: pool.globalLimit,
		ResetTimeout: pool.resetTO,
	})
	pool.breakers[backend] = cb
	return cb
}

func (pool *CircuitBreakerPool) Stats() map[string]interface{} {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	var breakers []map[string]interface{}
	for url, cb := range pool.breakers {
		breakers = append(breakers, map[string]interface{}{
			"backend": url,
			"state":   cb.state.String(),
			"stats":   cb.Stats(),
		})
	}

	return map[string]interface{}{
		"breakers": breakers,
	}
}

// =============================================================================
// Circuit Breaker Middleware
// =============================================================================

type CircuitMiddleware struct {
	pool *CircuitBreakerPool
}

func NewCircuitMiddleware(pool *CircuitBreakerPool) *CircuitMiddleware {
	return &CircuitMiddleware{pool: pool}
}

func (m *CircuitMiddleware) WrapHandler(backend string, handler http.Handler) http.Handler {
	cb := m.pool.GetBreaker(backend)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cb.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":{"code":"CIRCUIT_OPEN","message":"Service temporarily unavailable"}}`))
			return
		}

		// Create response wrapper to track status
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		handler.ServeHTTP(rw, r)

		// Record result
		if rw.statusCode >= 500 {
			cb.RecordFailure()
		} else {
			cb.RecordSuccess()
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}
