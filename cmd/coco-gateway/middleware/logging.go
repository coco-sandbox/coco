package middleware

import (
	"log"
	"net/http"
	"time"
)

type Logger struct {
	log *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		log: log.Default(),
	}
}

func (l *Logger) Logf(format string, v ...interface{}) {
	l.log.Printf(format, v...)
}

func Logging(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Logf(
				"%s %s %d %v",
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
			)
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
