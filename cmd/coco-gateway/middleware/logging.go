// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// StructuredLogEntry is a structured JSON log entry per spec.
type StructuredLogEntry struct {
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Component  string `json:"component"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	SandboxID  string `json:"sandbox_id,omitempty"`
}

type Logger struct {
	component string
}

func NewLogger() *Logger {
	return &Logger{
		component: "coco-gateway",
	}
}

func (l *Logger) Logf(format string, v ...interface{}) {
	entry := StructuredLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     "INFO",
		Component: l.component,
		Message:   fmt.Sprintf(format, v...),
	}
	if j, err := json.Marshal(entry); err == nil {
		log.Print(string(j))
	}
}

func Logging(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := r.Header.Get("X-Request-ID")

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			durationMs := duration.Milliseconds()

			entry := StructuredLogEntry{
				Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
				Level:      "INFO",
				Component:  "coco-gateway",
				Message:    "request completed",
				RequestID:  reqID,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: wrapped.statusCode,
				DurationMs: durationMs,
			}
			if j, err := json.Marshal(entry); err == nil {
				log.Print(string(j))
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
