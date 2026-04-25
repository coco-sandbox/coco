// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Command coco-core is the main API server for the Coco sandbox runtime.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/internal/config"
	"github.com/coco-sandbox/coco/internal/metrics"
	"github.com/coco-sandbox/coco/internal/types"
	"github.com/coco-sandbox/coco/pkg/cluster"
	"github.com/coco-sandbox/coco/pkg/api/handlers"
	"github.com/coco-sandbox/coco/pkg/middleware"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/visor"
)

type server struct {
	config   *config.Config
	mux      *http.ServeMux
	server   *http.Server
	store    *store.Store
	cluster  *cluster.Manager
	metrics  *metrics.Metrics
}

// AppState holds the application state
type AppState struct {
	sandboxes map[string]*types.Sandbox
	store     *store.Store
	cluster   *cluster.Manager
	metrics   *metrics.Metrics
	mu        struct {
		sync.RWMutex
	}
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := config.Load()

	s := &server{
		config: cfg,
		mux:    http.NewServeMux(),
	}

	if err := s.init(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	if err := s.start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	s.waitForSignal()
}

func (s *server) init() error {
	// Initialize store
	st, err := store.New(s.config.StoreDir)
	if err != nil {
		return err
	}
	s.store = st

	// Initialize cluster manager
	hostname, _ := os.Hostname()
	s.cluster = cluster.NewManager(hostname, "0.1.0")
	s.cluster.Start()

	// Initialize metrics
	s.metrics = metrics.New()

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.server = &http.Server{
		Addr:         s.config.ListenAddr,
		Handler:      s.wrapMiddleware(s.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return nil
}

func (s *server) start() error {
	log.Printf("coco-core starting on %s", s.config.ListenAddr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	return nil
}

func (s *server) setupRoutes() {
	// Health
	s.mux.HandleFunc("/health", handleHealth)
	s.mux.HandleFunc("/ready", handleReady)

	// Sandboxes
	s.mux.HandleFunc("/v1/sandboxes", handleSandboxes)
	s.mux.HandleFunc("/v1/sandboxes/", handleSandboxByID)

	// Cluster
	s.mux.HandleFunc("/cluster/nodes", handleClusterNodes)
	s.mux.HandleFunc("/cluster/health", handleClusterHealth)
}

func (s *server) wrapMiddleware(h http.Handler) http.Handler {
	h = middleware.RecoveryMiddleware(h)
	h = middleware.CORSMiddleware(h)
	h = middleware.TimeoutMiddleware(30 * time.Second)
	h = middleware.LoggingMiddleware(middleware.LogFunc())
	return h
}

func (s *server) waitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	s.cluster.Stop()
	s.store.Close()

	log.Println("coco-core stopped")
}

// HTTP Handlers

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"healthy":true,"version":"0.1.0"}`))
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ready":true,"version":"0.1.0"}`))
}

func handleSandboxes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"items":[],"total":0}`))
}

func handleSandboxByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not found"}`))
}

func handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"nodes":[],"total":0}`))
}

func handleClusterHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"healthy":true,"leader_id":"self"}`))
}

// SandboxService implementation

type sandboxService struct {
	store   *store.Store
	metrics *metrics.Metrics
}

func (s *sandboxService) Create(ctx context.Context, req *types.CreateSandboxRequest) (*types.Sandbox, error) {
	sb := &types.Sandbox{
		ID:        generateID(),
		Name:      req.Name,
		State:     types.SandboxStateCreating,
		Template:  req.Template,
		MemoryMB: req.MemoryMB,
		VCPUs:     req.VCPUs,
		Labels:    req.Labels,
	}

	if err := s.store.PutSandbox(sb); err != nil {
		return nil, err
	}

	return sb, nil
}

func (s *sandboxService) Get(ctx context.Context, id string) (*types.Sandbox, error) {
	return s.store.GetSandbox(id)
}

func (s *sandboxService) List(ctx context.Context) ([]*types.Sandbox, error) {
	return s.store.ListSandboxes()
}

func (s *sandboxService) Update(ctx context.Context, id string, sb *types.Sandbox) (*types.Sandbox, error) {
	return sb, s.store.PutSandbox(sb)
}

func (s *sandboxService) Delete(ctx context.Context, id string) error {
	return s.store.DeleteSandbox(id)
}

func (s *sandboxService) Pause(ctx context.Context, id string) error {
	return nil
}

func (s *sandboxService) Resume(ctx context.Context, id string) error {
	return nil
}

func (s *sandboxService) Fork(ctx context.Context, parentID string, req *types.ForkRequest) (*types.Sandbox, error) {
	return nil, nil
}

func (s *sandboxService) Hibernate(ctx context.Context, id string) error {
	return nil
}

func (s *sandboxService) ResumeHibernate(ctx context.Context, id string) error {
	return nil
}

func generateID() string {
	return "sb_" + time.Now().Format("20060102150405")
}

var _ handlers.SandboxService = (*sandboxService)(nil)
