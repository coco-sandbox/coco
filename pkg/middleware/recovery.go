package middleware

import (
	"fmt"
	"log"
	"net/http"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":{"code":"INTERNAL","message":"Internal server error"}}`)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func RecoveryWithLogger(logger func(string, ...interface{})) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger("Panic recovered: %v", err)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"error":{"code":"INTERNAL","message":"Internal server error"}}`)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
