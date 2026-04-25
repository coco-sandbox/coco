// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"bufio"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Compression returns a middleware that handles gzip compression for responses
func Compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Use GzipResponseWriter if client accepts gzip
		gz := &GzipResponseWriter{
			ResponseWriter: w,
			gzWriter:        nil,
		}

		// Create gzip writer
		gzWriter := gzip.NewWriter(w)
		gz.gzWriter = gzWriter

		// Set header
		w.Header().Set("Content-Encoding", "gzip")

		// Handle close
		defer gzWriter.Close()

		next.ServeHTTP(gz, r)
	})
}

// GzipResponseWriter wraps an http.ResponseWriter to gzip the response
type GzipResponseWriter struct {
	http.ResponseWriter
	gzWriter *gzip.Writer
}

func (w *GzipResponseWriter) Write(data []byte) (int, error) {
	return w.gzWriter.Write(data)
}

func (w *GzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// Flush implements http.Flusher for streaming responses
func (w *GzipResponseWriter) Flush() {
	w.gzWriter.Flush()
}

// Hijack implements http.Hijacker for websocket support
func (w *GzipResponseWriter) Hijack() (io.Writer, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, nil
}
