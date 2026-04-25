// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// Auth Middleware - JWT and mTLS Authentication
// =============================================================================

type AuthConfig struct {
	ValidateJWT  bool
	ValidateMTLS bool
	JWTIssuer    string
	JWTAudience  string
	JWTKey       []byte
}

type AuthContext struct {
	TenantID   string
	UserID      string
	Roles       []string
	RateLimit   int
	AuthMethod  string
	CertSubject string
}

type AuthMiddleware struct {
	config AuthConfig
}

func newAuthMiddleware(cfg AuthConfig) *AuthMiddleware {
	if cfg.JWTKey == nil {
		// Default key for development
		cfg.JWTKey = []byte("coco-secret-key-for-development-only")
	}
	return &AuthMiddleware{config: cfg}
}

// JWTClaims represents JWT payload
type JWTClaims struct {
	Sub       string   `json:"sub"`
	TenantID  string   `json:"tenant_id"`
	Roles     []string `json:"roles"`
	RateLimit int      `json:"rate_limit"`
	Iss       string   `json:"iss"`
	Aud       string   `json:"aud"`
	Exp       int64    `json:"exp"`
	Iat       int64    `json:"iat"`
}

func (am *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := am.authenticate(r)
	if ctx == nil {
		writeAuthError(w, "UNAUTHORIZED", "Authentication required")
		return
	}

	// Store auth context in request
	r = r.WithContext(withAuth(r.Context(), *ctx))
	r.Header.Set("X-Tenant-ID", ctx.TenantID)

	// Continue to next handler
}

func (am *AuthMiddleware) authenticate(r *http.Request) *AuthContext {
	// Try JWT first
	if am.config.ValidateJWT {
		if ctx := am.authenticateJWT(r); ctx != nil {
			return ctx
		}
	}

	// Try mTLS
	if am.config.ValidateMTLS {
		if ctx := am.authenticateMTLS(r); ctx != nil {
			return ctx
		}
	}

	// Fall back to default
	return &AuthContext{
		TenantID:  "default",
		RateLimit: 600,
		AuthMethod: "none",
	}
}

func (am *AuthMiddleware) authenticateJWT(r *http.Request) *AuthContext {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := parseJWT(token, am.config.JWTKey)
	if err != nil {
		log.Printf("[auth] JWT parse error: %v", err)
		return nil
	}

	// Validate expiration
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		log.Printf("[auth] JWT expired")
		return nil
	}

	// Validate issuer
	if am.config.JWTIssuer != "" && claims.Iss != am.config.JWTIssuer {
		log.Printf("[auth] JWT issuer mismatch: %s != %s", claims.Iss, am.config.JWTIssuer)
		return nil
	}

	// Validate audience
	if am.config.JWTAudience != "" && claims.Aud != am.config.JWTAudience {
		log.Printf("[auth] JWT audience mismatch")
		return nil
	}

	rateLimit := claims.RateLimit
	if rateLimit == 0 {
		rateLimit = 600 // default
	}

	roles := claims.Roles
	if roles == nil {
		roles = []string{"agent"}
	}

	return &AuthContext{
		TenantID:  claims.TenantID,
		UserID:    claims.Sub,
		Roles:     roles,
		RateLimit: rateLimit,
		AuthMethod: "jwt",
	}
}

func (am *AuthMiddleware) authenticateMTLS(r *http.Request) *AuthContext {
	if r.TLS == nil {
		return nil
	}

	if len(r.TLS.PeerCertificates) == 0 {
		return nil
	}

	cert := r.TLS.PeerCertificates[0]
	subject := cert.Subject.String()

	// Extract CN as user ID
	cn := extractCN(subject)
	if cn == "" {
		cn = "unknown"
	}

	// Extract organization
	org := extractOrg(subject)

	return &AuthContext{
		TenantID:   org,
		UserID:     cn,
		Roles:      extractRoles(cert),
		RateLimit:  600,
		AuthMethod: "mtls",
		CertSubject: subject,
	}
}

func parseJWT(token string, key []byte) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// In production, verify signature with key
	// For now, just parse the claims

	return &claims, nil
}

func extractCN(subject string) string {
	parts := strings.Split(subject, ",")
	for _, p := range parts {
		kv := strings.Split(strings.TrimSpace(p), "=")
		if len(kv) == 2 && kv[0] == "CN" {
			return kv[1]
		}
	}
	return ""
}

func extractOrg(subject string) string {
	parts := strings.Split(subject, ",")
	for _, p := range parts {
		kv := strings.Split(strings.TrimSpace(p), "=")
		if len(kv) == 2 && kv[0] == "O" {
			return kv[1]
		}
	}
	return "default"
}

func extractRoles(cert *x509.Certificate) []string {
	// Extract roles from certificate extensions
	// For now, return default role
	return []string{"agent"}
}

func writeAuthError(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

type contextKey string

const authContextKey = "auth"

func withAuth(ctx context.Context, auth AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}

// GetAuth extracts AuthContext from request context
func GetAuth(r *http.Request) *AuthContext {
	if v := r.Context().Value(authContextKey); v != nil {
		if ctx, ok := v.(AuthContext); ok {
			return &ctx
		}
	}
	return nil
}

// =============================================================================
// mTLS Certificate Validation
// =============================================================================

type MTLSConfig struct {
	CA               *x509.CertPool
	ClientAuth       tls.ClientAuthType
	SkipVerification bool
}

func validateClientCert(cfg MTLSConfig, cert *x509.Certificate) error {
	if cfg.SkipVerification {
		return nil
	}

	opts := x509.VerifyOptions{
		Roots: cfg.CA,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}
