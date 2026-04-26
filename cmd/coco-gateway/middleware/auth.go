// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/coco-sandbox/coco/pkg/api"
)

var ErrUnauthorized = errors.New("unauthorized")

type contextKey string

const userContextKey contextKey = "user"
const projectContextKey contextKey = "project"

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userContextKey).(string)
	return user, ok
}

func WithProject(ctx context.Context, project string) context.Context {
	return context.WithValue(ctx, projectContextKey, project)
}

func ProjectFromContext(ctx context.Context) (string, bool) {
	project, ok := ctx.Value(projectContextKey).(string)
	return project, ok
}

type Authenticator interface {
	Authenticate(r *http.Request) (string, error)
}

type TokenAuth struct {
	tokens map[string]string
}

func NewTokenAuth() *TokenAuth {
	return &TokenAuth{
		tokens: make(map[string]string),
	}
}

func (t *TokenAuth) AddToken(token, user string) {
	t.tokens[token] = user
}

func (t *TokenAuth) Authenticate(r *http.Request) (string, error) {
	if len(t.tokens) == 0 {
		return "anonymous", nil
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrUnauthorized
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrUnauthorized
	}

	token := parts[1]
	user, ok := t.tokens[token]
	if !ok {
		return "", ErrUnauthorized
	}

	return user, nil
}

// Auth returns a middleware that authenticates requests.
func Auth(auth Authenticator, audit *AuditLogger) func(http.Handler) http.Handler {
	skipPaths := map[string]bool{
		"/health":       true,
		"/health/live":  true,
		"/health/ready": true,
		"/metrics":      true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			sourceIP := GetClientIP(r)
			user, err := auth.Authenticate(r)
			if err != nil {
				if audit != nil {
					audit.LogAuthFailure("", sourceIP, err.Error())
				}
				api.WriteUnauthorized(w, err.Error())
				return
			}

			if audit != nil {
				audit.LogAuthSuccess(user, sourceIP)
			}

			ctx := r.Context()
			ctx = WithUser(ctx, user)

			if project := r.Header.Get("X-Coco-Project"); project != "" {
				ctx = WithProject(ctx, project)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
