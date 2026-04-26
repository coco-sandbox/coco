// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

var ErrUnauthorized = errors.New("unauthorized")

type contextKey string

const userContextKey contextKey = "user"

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userContextKey).(string)
	return user, ok
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

func Auth(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := auth.Authenticate(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			r = r.WithContext(WithUser(r.Context(), user))
			next.ServeHTTP(w, r)
		})
	}
}