// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/cmd/coco-gateway/middleware"
	"github.com/coco-sandbox/coco/pkg/config"
	"github.com/coco-sandbox/coco/pkg/metrics"
	cocoauth "github.com/coco-sandbox/coco/pkg/middleware/auth"
	"github.com/coco-sandbox/coco/pkg/visor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func timestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := config.Load()

	log.Printf("coco-gateway starting (listen_addr=%s, master_addr=%s)", cfg.ListenAddr, cfg.MasterAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx, cfg); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	metrics.Register()

	tokenAuth := middleware.NewTokenAuth()
	for token, user := range cfg.APIKeys {
		tokenAuth.AddToken(token, user)
	}

	auditLogger := middleware.NewAuditLogger()

	keyStore := cocoauth.NewInMemoryStore()

	for token, user := range cfg.APIKeys {
		key := &cocoauth.APIKey{
			ID:        user,
			Name:      user,
			KeyHash:   cocoauth.HashKey(token),
			Role:      cocoauth.RoleOperator,
			Scopes:    []cocoauth.Scope{cocoauth.ScopeSandboxCreate, cocoauth.ScopeSandboxRead, cocoauth.ScopeSandboxWrite, cocoauth.ScopeSandboxDelete},
			CreatedAt: timestampNow(),
			Enabled:   true,
		}
		keyStore.CreateKey(context.Background(), key)
	}

	gw := NewGatewayServer(cfg.MasterAddr, nil)
	vp := visor.NewPool(visor.SocketPath, 10)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health/live", http.StatusMovedPermanently)
	})

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		checks := map[string]string{
			"gateway": "ok",
		}
		checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		_, err := gw.GetClusterInfo(checkCtx)
		if err != nil {
			checks["master"] = "unreachable"
		} else {
			checks["master"] = "ok"
		}

		status := "ready"
		for _, v := range checks {
			if v != "ok" {
				status = "degraded"
				break
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": status,
			"checks": checks,
		})
	})

	mux.Handle("/metrics", metrics.Handler())

	registerRoutes(mux, gw, tokenAuth, vp, keyStore)

	handler := middleware.RecoveryMiddleware()(mux)
	handler = middleware.CORS(middleware.DefaultCORSConfig())(handler)
	if cfg.RateLimitEnabled {
		rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, int64(cfg.RateLimitBurst), time.Second)
		handler = middleware.RateLimit(rateLimiter)(handler)
	}
	handler = middleware.Auth(tokenAuth, auditLogger)(handler)
	handler = middleware.Tracing(handler)
	handler = middleware.Logging(middleware.NewLogger())(handler)

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	log.Println("Gateway stopped")
	return nil
}
