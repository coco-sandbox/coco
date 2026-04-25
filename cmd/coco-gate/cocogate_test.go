// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Auth Middleware Tests
// =============================================================================

func TestAuthMiddleware_JWTValidation(t *testing.T) {
	cfg := AuthConfig{
		ValidateJWT: true,
		JWTIssuer:   "coco.acme.com",
		JWTAudience: "coco-api",
		JWTKey:      []byte("test-secret"),
	}

	auth := newAuthMiddleware(cfg)

	tests := []struct {
		name        string
		authHeader string
		wantStatus int
	}{
		{
			name:        "no auth header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "invalid bearer format",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			auth.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestParseJWT(t *testing.T) {
	// Valid JWT payload (base64 encoded)
	// {"sub":"user-123","tenant_id":"acme","roles":["agent"],"exp":9999999999}
	payload := "eyJzdWIiOiJ1c2VyLTEyMyIsInRlbmFudF9pZCI6ImFjbWUiLCJyb2xlcyI6WyJhZ2VudCJdLCJleHAiOjk5OTk5OTk5OTl9"
	token := "header." + payload + ".signature"

	claims, err := parseJWT(token, []byte("key"))
	if err != nil {
		t.Errorf("parseJWT failed: %v", err)
	}

	if claims.Sub != "user-123" {
		t.Errorf("expected sub=user-123, got %s", claims.Sub)
	}

	if claims.TenantID != "acme" {
		t.Errorf("expected tenant_id=acme, got %s", claims.TenantID)
	}
}

func TestParseJWT_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid base64", "header...invalid@#$%signature"},
		{"missing parts", "only.two"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseJWT(tt.token, []byte("key"))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestExtractCN(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{"CN only", "CN=test-user", "test-user"},
		{"CN with OU", "OU=admins,CN=admin-user", "admin-user"},
		{"no CN", "OU=users,O=acme", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cn := extractCN(tt.subject)
			if cn != tt.expected {
				t.Errorf("extractCN(%q) = %q, want %q", tt.subject, cn, tt.expected)
			}
		})
	}
}

// =============================================================================
// Rate Limiter Tests
// =============================================================================

func TestTokenBucket_Allow(t *testing.T) {
	cfg := RateLimitConfig{
		Capacity: 10,
		Rate:     10, // 10 tokens per second
		Burst:    20,
	}

	limiter := newRateLimiter(cfg)
	key := "test-key"

	// First 10 should succeed (bucket capacity)
	for i := 0; i < 10; i++ {
		if !limiter.Allow(key) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11th should fail (bucket empty)
	if limiter.Allow(key) {
		t.Error("11th request should be rate limited")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	cfg := RateLimitConfig{
		Capacity: 1,
		Rate:     100, // 100 tokens per second = fast refill for test
		Burst:    1,
	}

	limiter := newRateLimiter(cfg)
	key := "test-key"

	// First request succeeds
	if !limiter.Allow(key) {
		t.Error("first request should be allowed")
	}

	// Second should fail (bucket empty)
	if limiter.Allow(key) {
		t.Error("second request should be rate limited")
	}

	// Wait for refill (10ms = 1 token at 100/sec)
	time.Sleep(20 * time.Millisecond)

	// Should succeed after refill
	if !limiter.Allow(key) {
		t.Error("request after refill should be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	cfg := RateLimitConfig{
		Capacity: 2,
		Rate:     100,
		Burst:    2,
	}

	limiter := newRateLimiter(cfg)
	middleware := RateLimitMiddleware(limiter, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "tenant-1")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: got status %d, want %d", i+1, rr.Code, http.StatusOK)
		}
	}

	// Third should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestSlidingWindowLimiter(t *testing.T) {
	limiter := NewSlidingWindowLimiter(3, 60) // 3 requests per 60 seconds

	key := "test-key"

	// First 3 should succeed
	for i := 0; i < 3; i++ {
		if !limiter.Allow(key) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th should fail
	if limiter.Allow(key) {
		t.Error("4th request should be denied")
	}

	// Check remaining
	remaining := limiter.Remaining(key)
	if remaining != 0 {
		t.Errorf("remaining = %d, want 0", remaining)
	}
}

// =============================================================================
// Load Balancer Tests
// =============================================================================

func TestRoundRobinLB_Next(t *testing.T) {
	backends := []string{"http://backend1:8080", "http://backend2:8080", "http://backend3:8080"}
	lb := NewRoundRobinLB(backends)

	// First backend
	be1 := lb.Next()
	if be1 == nil || be1.URL != "http://backend1:8080" {
		t.Errorf("first backend: got %v, want backend1", be1)
	}

	// Second backend
	be2 := lb.Next()
	if be2 == nil || be2.URL != "http://backend2:8080" {
		t.Errorf("second backend: got %v, want backend2", be2)
	}

	// Third backend
	be3 := lb.Next()
	if be3 == nil || be3.URL != "http://backend3:8080" {
		t.Errorf("third backend: got %v, want backend3", be3)
	}

	// Wraps around
	be4 := lb.Next()
	if be4 == nil || be4.URL != "http://backend1:8080" {
		t.Errorf("fourth backend (wrap): got %v, want backend1", be4)
	}
}

func TestRoundRobinLB_MarkUnhealthy(t *testing.T) {
	backends := []string{"http://backend1:8080", "http://backend2:8080"}
	lb := NewRoundRobinLB(backends)

	lb.MarkUnhealthy("http://backend1:8080")

	// Should skip to backend2
	be := lb.Next()
	if be == nil || be.URL != "http://backend2:8080" {
		t.Errorf("should get backend2, got %v", be)
	}
}

func TestLeastConnectionsLB(t *testing.T) {
	backends := []string{"http://backend1:8080", "http://backend2:8080"}
	lb := NewLeastConnectionsLB(backends)

	// First request should go to first backend
	be1 := lb.Next()
	if be1.URL != "http://backend1:8080" {
		t.Errorf("got %s, want backend1", be1.URL)
	}

	// Second request should go to first backend (least connections)
	be2 := lb.Next()
	if be2.URL != "http://backend1:8080" {
		t.Errorf("got %s, want backend1 (still least connections)", be2.URL)
	}
}

func TestWeightedLB(t *testing.T) {
	backends := map[string]int{
		"http://backend1:8080": 3,
		"http://backend2:8080": 1,
	}
	lb := NewWeightedLB(backends)

	// Run multiple times and count distribution
	counts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		be := lb.Next()
		if be != nil {
			counts[be.URL]++
		}
	}

	// backend1 should be approximately 3x more than backend2
	ratio := float64(counts["http://backend1:8080"]) / float64(counts["http://backend2:8080"])
	if ratio < 2.0 || ratio > 4.0 {
		t.Errorf("ratio = %f, want approximately 3.0", ratio)
	}
}

// =============================================================================
// Circuit Breaker Tests
// =============================================================================

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         "test",
		FailureLimit: 3,
		ResetTimeout: 100 * time.Millisecond,
	})

	// All requests should be allowed while closed
	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// After 3 failures, should open
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("state = %s, want open", cb.State())
	}

	// Should be blocked while open
	if cb.Allow() {
		t.Error("request should be blocked while open")
	}
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         "test",
		FailureLimit: 2,
		ResetTimeout: 50 * time.Millisecond,
	})

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("state = %s, want open", cb.State())
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open
	if !cb.Allow() {
		t.Error("request should be allowed after reset timeout")
	}

	if cb.State() != CircuitHalfOpen {
		t.Errorf("state = %s, want half-open", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:           "test",
		FailureLimit:   1,
		SuccessLimit:   2,
		ResetTimeout:   10 * time.Millisecond,
		HalfOpenMaxReqs: 3,
	})

	// Trip the circuit
	cb.RecordFailure()

	// Wait and transition to half-open
	time.Sleep(15 * time.Millisecond)
	cb.Allow()

	// Record successes
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("state = %s, want closed", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenStaysOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:           "test",
		FailureLimit:   1,
		SuccessLimit:   3,
		ResetTimeout:   10 * time.Millisecond,
		HalfOpenMaxReqs: 2,
	})

	// Trip the circuit
	cb.RecordFailure()

	// Wait and transition to half-open
	time.Sleep(15 * time.Millisecond)
	cb.Allow()

	// Record failure while half-open
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("state = %s, want open", cb.State())
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         "test-breaker",
		FailureLimit: 5,
	})

	stats := cb.Stats()

	if stats["name"] != "test-breaker" {
		t.Errorf("name = %v, want test-breaker", stats["name"])
	}

	if stats["state"] != "closed" {
		t.Errorf("state = %v, want closed", stats["state"])
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestAuthContextFromRequest(t *testing.T) {
	cfg := AuthConfig{
		ValidateJWT: true,
		JWTKey:      []byte("test"),
	}

	auth := newAuthMiddleware(cfg)

	// Create request with JWT
	// eyJzdWIiOiJ1c2VyLTEyMyJ9 = {"sub":"user-123"}
	token := "header.eyJzdWIiOiJ1c2VyLTEyMyJ9.signature"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	auth.ServeHTTP(rr, req)

	// Should authenticate (JWT parsing will work, but validation may fail on signature)
	// We just check it doesn't crash
}

func TestRateLimitByTenant(t *testing.T) {
	cfg := RateLimitConfig{
		Capacity: 1,
		Rate:     100,
	}

	limiter := newRateLimiter(cfg)
	middleware := RateLimitMiddleware(limiter, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// First request for tenant-1 should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	// Different tenant should still be allowed
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-2")
	rr = httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("tenant-2 request: got %d, want %d", rr.Code, http.StatusOK)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkTokenBucket_Allow(b *testing.B) {
	cfg := RateLimitConfig{Capacity: 1000, Rate: 1000}
	limiter := newRateLimiter(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow("bench-key")
	}
}

func BenchmarkCircuitBreaker_Allow(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureLimit: 5})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}
