package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"coco/pkg/api"
)

type Recovery struct {
	logger *log.Logger
}

func NewRecovery() *Recovery {
	return &Recovery{
		logger: log.Default(),
	}
}

func (r *Recovery) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			r.logger.Printf("panic recovered: %v\n%s", err, debug.Stack())
			api.WriteInternalError(w, "internal server error")
		}
	}()

	w.WriteHeader(http.StatusOK)
}

func RecoveryMiddleware() func(http.Handler) http.Handler {
	rec := NewRecovery()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					rec.logger.Printf("panic recovered: %v\n%s", err, debug.Stack())
					api.WriteInternalError(w, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
