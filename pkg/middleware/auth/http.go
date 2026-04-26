// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/coco-sandbox/coco/pkg/api"
)

var (
	ErrMissingAuthHeader = errors.New("missing Authorization header")
	ErrInvalidAuthScheme = errors.New("invalid Authorization scheme")
)

type Middleware struct {
	authenticator *Authenticator
	skipPaths    map[string]bool
}

type MiddlewareOption func(*Middleware)

func WithSkipPath(path string) MiddlewareOption {
	return func(m *Middleware) {
		m.skipPaths[path] = true
	}
}

func NewMiddleware(store KeyStore, opts ...MiddlewareOption) *Middleware {
	m := &Middleware{
		authenticator: NewAuthenticator(store),
		skipPaths: map[string]bool{
			"/health":       true,
			"/health/live":  true,
			"/health/ready": true,
			"/metrics":      true,
		},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.skipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		key, err := m.extractAndValidateKey(r)
		if err != nil {
			api.WriteError(w, api.ErrPermissionDenied, err.Error(), "", http.StatusUnauthorized)
			return
		}

		ctx := WithAuth(r.Context(), key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) extractAndValidateKey(r *http.Request) (*APIKey, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		if m.authenticator != nil {
			return nil, ErrMissingAuthHeader
		}
		return nil, nil
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidAuthScheme
	}

	scheme := strings.ToLower(parts[0])
	if scheme != "bearer" {
		return nil, ErrInvalidAuthScheme
	}

	rawKey := parts[1]
	if rawKey == "" {
		return nil, ErrInvalidKey
	}

	ctx := r.Context()
	key, err := m.authenticator.ValidateKey(ctx, rawKey)
	if err != nil {
		return nil, err
	}

	return key, nil
}
