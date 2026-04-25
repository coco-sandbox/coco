// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type key int

const RequestIDKey key = 0

// RequestID is a middleware that generates a unique request ID for each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID already exists (from load balancer or client)
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate new request ID (16 bytes = 32 hex chars)
			b := make([]byte, 16)
			_, err := rand.Read(b)
			if err == nil {
				requestID = hex.EncodeToString(b)
			} else {
				// Fallback to simple ID
				requestID = "unknown"
			}
		}

		// Store in context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Set response header
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
