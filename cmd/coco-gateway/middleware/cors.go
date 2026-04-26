package middleware

import (
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

func CORS(config *CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if r.Method == "OPTIONS" {
				handlePreflight(w, r, config)
				return
			}

			if len(config.AllowOrigins) > 0 && config.AllowOrigins[0] != "*" {
				allowed := false
				for _, o := range config.AllowOrigins {
					if o == origin {
						allowed = true
						break
					}
				}
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			} else if config.AllowOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func handlePreflight(w http.ResponseWriter, r *http.Request, config *CORSConfig) {
	origin := r.Header.Get("Origin")

	if len(config.AllowOrigins) > 0 && config.AllowOrigins[0] != "*" {
		allowed := false
		for _, o := range config.AllowOrigins {
			if o == origin {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			return
		}
	} else if config.AllowOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	method := r.Header.Get("Access-Control-Request-Method")
	if method != "" {
		allowed := false
		for _, m := range config.AllowMethods {
			if m == method {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Method", method)
		}
	}

	headers := r.Header.Get("Access-Control-Request-Headers")
	if headers != "" {
		w.Header().Set("Access-Control-Allow-Headers", headers)
	}

	if config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if config.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
	}

	w.WriteHeader(http.StatusNoContent)
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == origin || o == "*" {
			return true
		}
		if strings.HasPrefix(o, "*.") {
			suffix := o[1:]
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}
	return false
}
