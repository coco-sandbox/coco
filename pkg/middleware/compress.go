package middleware

import (
	"net/http"
	"strings"
)

func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")

		gzipWriter := &gzipResponseWriter{ResponseWriter: w}
		next.ServeHTTP(gzipWriter, r)
		gzipWriter.Close()
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

func (w *gzipResponseWriter) Close() {
}
