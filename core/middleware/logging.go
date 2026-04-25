// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// RequestLogEntry represents a structured log entry for HTTP requests
type RequestLogEntry struct {
	Timestamp   string `json:"timestamp"`
	RequestID   string `json:"request_id,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	StatusCode  int    `json:"status_code"`
	DurationMs  int64  `json:"duration_ms"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent,omitempty"`
	Error       string `json:"error,omitempty"`
	SandboxID   string `json:"sandbox_id,omitempty"`
}

// RequestLogging returns a middleware that logs all HTTP requests in JSON format
func RequestLogging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &StatusCapturingWriter{
				ResponseWriter: w,
				statusCode:      http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Build log entry
			entry := RequestLogEntry{
				Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
				RequestID:   requestID,
				Method:      r.Method,
				Path:        r.URL.Path,
				StatusCode:  wrapped.statusCode,
				DurationMs: duration.Milliseconds(),
				ClientIP:    clientIP(r),
				UserAgent:   r.UserAgent(),
			}

			// Log as JSON
			data, _ := json.Marshal(entry)
			logger.Println(string(data))
		})
	}
}

// StatusCapturingWriter wraps an http.ResponseWriter to capture status code
type StatusCapturingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *StatusCapturingWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures the status code if not already set
func (w *StatusCapturingWriter) Write(data []byte) (int, error) {
	if w.statusCode == http.StatusOK {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

// clientIP extracts the client IP from the request, considering proxies
func clientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// SandboxIDFromPath extracts sandbox ID from request path
func SandboxIDFromPath(path string) string {
	// Simple extraction - in production use proper routing
	const prefix = "/v1/sandboxes/"
	if len(path) > len(prefix) {
		remaining := path[len(prefix):]
		if idx := strings.Index(remaining, "/"); idx > 0 {
			return remaining[:idx]
		}
		return remaining
	}
	return ""
}
