// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Auth Types
// =============================================================================

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleOperator Role = "operator"
	RoleAgent   Role = "agent"
	RoleReadonly Role = "readonly"
)

// RBAC permission matrix
var rolePermissions = map[Role][]string{
	RoleAdmin: {
		"sandbox:create", "sandbox:read", "sandbox:write", "sandbox:delete",
		"sandbox:exec", "sandbox:fork", "sandbox:hibernate", "sandbox:checkpoint",
		"sandbox:replay",
		"node:read", "node:write",
		"template:read", "template:write",
		"admin:write",
	},
	RoleOperator: {
		"sandbox:create", "sandbox:read", "sandbox:write", "sandbox:delete",
		"sandbox:exec", "sandbox:fork", "sandbox:hibernate", "sandbox:checkpoint",
		"sandbox:replay",
		"node:read",
		"template:read",
	},
	RoleAgent: {
		"sandbox:create", "sandbox:read", "sandbox:write",
		"sandbox:exec", "sandbox:fork",
		"template:read",
	},
	RoleReadonly: {
		"sandbox:read",
		"node:read",
		"template:read",
	},
}

// =============================================================================
// API Key Auth
// =============================================================================

type APIKey struct {
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	TenantID string   `json:"tenant_id"`
	Roles    []Role   `json:"roles"`
	Expires  int64    `json:"expires,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (k *APIKey) HasPermission(perm string) bool {
	for _, role := range k.Roles {
		perms := rolePermissions[role]
		for _, p := range perms {
			if p == perm || p == "*" {
				return true
			}
		}
	}
	return false
}

// apiKeys maps API key hash -> APIKey
var apiKeys = make(map[string]*APIKey)
var apiKeysMu sync.RWMutex

func validateAPIKey(hashedKey string) (*APIKey, error) {
	apiKeysMu.RLock()
	defer apiKeysMu.RUnlock()

	key, ok := apiKeys[hashedKey]
	if !ok {
		return nil, fmt.Errorf("invalid API key")
	}

	if key.Expires > 0 && key.Expires < time.Now().Unix() {
		return nil, fmt.Errorf("API key expired")
	}

	return key, nil
}

func hashAPIKey(rawKey string) string {
	// In production, use proper key derivation (Argon2, scrypt, etc.)
	// This is a placeholder using SHA256 for demonstration
	// TODO: Implement proper key hashing
	return rawKey // placeholder
}

// =============================================================================
// Rate Limiting
// =============================================================================

type rateLimiter struct {
	tokens    map[string]*tokenBucket
	mu        sync.RWMutex
	rate      float64   // tokens per second
	burst     int       // max burst size
	expiresAt int64     // unix timestamp when this limiter expires
}

type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
	mu        sync.Mutex
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	return &rateLimiter{
		tokens: make(map[string]*tokenBucket),
		rate:   rate,
		burst:  burst,
	}
}

func (rl *rateLimiter) Allow(tenantID string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.tokens[tenantID]
	if !exists {
		bucket = &tokenBucket{
			tokens:    float64(rl.burst),
			lastCheck: time.Now(),
		}
		rl.tokens[tenantID] = bucket
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastCheck).Seconds()
	bucket.lastCheck = now

	// Add tokens based on elapsed time
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}

	// Check if request can be served
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}

	// Calculate retry-after in seconds
	retryAfter := int((1 - bucket.tokens) / rl.rate)
	if retryAfter < 1 {
		retryAfter = 1
	}

	return false, retryAfter
}

// =============================================================================
// Global Rate Limiters
// =============================================================================

// Default rate limiter: 100 requests/second, burst of 200
var defaultLimiter = newRateLimiter(100, 200)

// Per-tenant rate limiters
var tenantLimiters = make(map[string]*rateLimiter)
var tenantLimitersMu sync.RWMutex

func getTenantLimiter(tenantID string) *rateLimiter {
	tenantLimitersMu.RLock()
	limiter, exists := tenantLimiters[tenantID]
	tenantLimitersMu.RUnlock()

	if exists {
		return limiter
	}

	tenantLimitersMu.Lock()
	defer tenantLimitersMu.Unlock()

	if limiter, exists = tenantLimiters[tenantID]; exists {
		return limiter
	}

	limiter = newRateLimiter(50, 100) // Default: 50 rps, burst 100
	tenantLimiters[tenantID] = limiter
	return limiter
}

// =============================================================================
// Middleware Functions
// =============================================================================

// AuthMiddleware validates API key and sets tenant context
func AuthMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health/ready/metrics endpoints
		path := r.URL.Path
		if path == "/health" || path == "/ready" || path == "/metrics" {
			handler.ServeHTTP(w, r)
			return
		}

		// Get API key from header
		rawKey := r.Header.Get("X-API-Key")
		if rawKey == "" {
			// Also check Authorization header
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				rawKey = auth[7:]
			}
		}

		if rawKey == "" {
			writeUnauthorizedError(w)
			// Audit log auth failure - missing key
			auditLogger.LogAuthAction(AuditResultFailure, r, "api_key", fmt.Errorf("missing API key"))
			return
		}

		hashedKey := hashAPIKey(rawKey)
		apiKey, err := validateAPIKey(hashedKey)
		if err != nil {
			writeUnauthorizedError(w)
			// Audit log auth failure
			auditLogger.LogAuthAction(AuditResultFailure, r, "api_key", err)
			return
		}

		// Audit log auth success
		auditLogger.LogAuthAction(AuditResultSuccess, r, "api_key", nil)

		// Set API key in context for downstream handlers
		ctx := WithAPIKey(r.Context(), apiKey)
		ctx = WithTenantID(ctx, apiKey.TenantID)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimitMiddleware applies rate limiting per tenant
func RateLimitMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health/ready/metrics endpoints
		path := r.URL.Path
		if path == "/health" || path == "/ready" || path == "/metrics" {
			handler.ServeHTTP(w, r)
			return
		}

		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			tenantID = "default"
		}

		limiter := getTenantLimiter(tenantID)
		allowed, retryAfter := limiter.Allow(tenantID)

		if !allowed {
			writeRateLimitedError(w, retryAfter)
			// Audit log rate limit event
			auditLogger.LogRateLimitAction(r, tenantID)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// RequirePermission checks if the current API key has the required permission
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := APIKeyFromContext(r.Context())
			if apiKey == nil {
				writeUnauthorizedError(w)
				return
			}

			if !apiKey.HasPermission(perm) {
				writeForbiddenError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// mTLS Support (Optional)
// =============================================================================

// TLSCert represents TLS certificate info
type TLSCert struct {
	Subject   string `json:"subject"`
	Issuer    string `json:"issuer"`
	NotBefore int64  `json:"not_before"`
	NotAfter  int64  `json:"not_after"`
	Serial    string `json:"serial"`
}

// GetTLSCertFromRequest extracts TLS cert info from request
func GetTLSCertFromRequest(r *http.Request) *TLSCert {
	if r.TLS == nil {
		return nil
	}

	// Get peer certificates
	if len(r.TLS.PeerCertificates) == 0 {
		return nil
	}

	cert := r.TLS.PeerCertificates[0]
	return &TLSCert{
		Subject:   cert.Subject.String(),
		Issuer:    cert.Issuer.String(),
		NotBefore: cert.NotBefore.Unix(),
		NotAfter:  cert.NotAfter.Unix(),
		Serial:    cert.SerialNumber.String(),
	}
}

// IsMTLSEnabled checks if mTLS is configured
func IsMTLSEnabled() bool {
	// TODO: Check config for mTLS setting
	return false
}

// =============================================================================
// API Key Management (for testing/admin)
// =============================================================================

func RegisterAPIKey(rawKey, name, tenantID string, roles []Role, expires int64) string {
	apiKeysMu.Lock()
	defer apiKeysMu.Unlock()

	hashedKey := hashAPIKey(rawKey)
	apiKeys[hashedKey] = &APIKey{
		Key:       rawKey,
		Name:      name,
		TenantID:  tenantID,
		Roles:     roles,
		Expires:   expires,
		CreatedAt: time.Now(),
	}

	return rawKey
}

func CompareAPIKey(stored, provided string) bool {
	return subtle.ConstantTimeCompare([]byte(stored), []byte(provided)) == 1
}