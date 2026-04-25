// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout returns a middleware that adds a timeout to the request context
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TimeoutHandler is an http.Handler that wraps another handler with a timeout
type TimeoutHandler struct {
	handler  http.Handler
	timeout  time.Duration
}

func (h *TimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		h.handler.ServeHTTP(w, r)
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Timeout - write timeout response
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte(`{"error":"request timeout"}`))
	case <-done:
		// Completed normally
	}
}
