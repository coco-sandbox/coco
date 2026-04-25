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
	"strings"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/internal/config"
	"github.com/coco-sandbox/coco/internal/metrics"
	"github.com/coco-sandbox/coco/internal/template"
	"github.com/coco-sandbox/coco/internal/types"
	"github.com/coco-sandbox/coco/pkg/api/handlers"
	"github.com/coco-sandbox/coco/pkg/cluster"
	"github.com/coco-sandbox/coco/pkg/middleware"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/visor"
)

type server struct {
	config      *config.Config
	mux         *http.ServeMux
	server      *http.Server
	store       *store.Store
	cluster     *cluster.Manager
	metrics     *metrics.Metrics
	templateMgr *template.Manager
	visorPool   *visor.Pool
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
	// Initialize template manager
	s.templateMgr = template.NewManager(s.config.Templates)

	// Initialize store
	st, err := store.New(s.config.StoreDir)
	if err != nil {
		return err
	}
	s.store = st

	// Initialize visor client pool
	s.visorPool = visor.NewPool(visor.SocketPath, 10)

	// Initialize cluster manager
	hostname, _ := os.Hostname()
	s.cluster = cluster.NewManager(hostname, "0.2.0")
	s.cluster.Start()

	// Initialize metrics
	s.metrics = metrics.New()

	// Setup routes with wired handlers
	s.setupRoutes()

	// Create HTTP server with streaming support
	s.server = &http.Server{
		Addr:         s.config.ListenAddr,
		Handler:      s.wrapMiddleware(s.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second, // Increased for streaming
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

	// Create handler instances with dependencies
	sandboxHandler := handlers.NewSandboxHandler(s)
	execHandler := handlers.NewExecHandler()
	templateHandler := handlers.NewTemplateHandler(s.templateMgr)

	// Sandboxes
	s.mux.HandleFunc("/v1/sandboxes", handleSandboxes(sandboxHandler))
	s.mux.HandleFunc("/v1/sandboxes/", handleSandboxByID(sandboxHandler, execHandler))

	// Templates
	s.mux.HandleFunc("/v1/templates", handleTemplates(templateHandler))
	s.mux.HandleFunc("/v1/templates/", handleTemplateByID(templateHandler))

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
	s.visorPool.Close()
	s.store.Close()

	log.Println("coco-core stopped")
}

// HTTP Handlers

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"healthy":true,"version":"0.2.0"}`))
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ready":true,"version":"0.2.0"}`))
}

func handleSandboxes(h *handlers.SandboxHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			h.HandleCreate(w, r)
		case "GET":
			h.HandleList(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleSandboxByID(sandboxHandler *handlers.SandboxHandler, execHandler *handlers.ExecHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := extractSandboxID(r.URL.Path)
		if id == "" {
			http.Error(w, "missing sandbox ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			sandboxHandler.HandleGet(w, r, id)
		case "DELETE":
			sandboxHandler.HandleDelete(w, r, id)
		case "POST":
			if strings.HasSuffix(r.URL.Path, "/pause") {
				sandboxHandler.HandlePause(w, r, id)
			} else if strings.HasSuffix(r.URL.Path, "/resume") {
				sandboxHandler.HandleResume(w, r, id)
			} else if strings.HasSuffix(r.URL.Path, "/exec") {
				execHandler.HandleExec(w, r, id)
			} else if strings.HasSuffix(r.URL.Path, "/streaming-exec") {
				execHandler.HandleStreamingExec(w, r, id)
			} else {
				http.Error(w, "unknown action", http.StatusNotFound)
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleTemplates(h *handlers.TemplateHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			h.HandleList(w, r)
		case "POST":
			h.HandleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleTemplateByID(h *handlers.TemplateHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := extractTemplateID(r.URL.Path)
		if id == "" {
			http.Error(w, "missing template ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			h.HandleGet(w, r, id)
		case "DELETE":
			h.HandleDelete(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
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
		MemoryMB:  req.MemoryMB,
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
var _ handlers.TemplateService = (*template.Manager)(nil)

// ID extraction helpers

func extractSandboxID(path string) string {
	// Expected format: /v1/sandboxes/<id>[/action]
	path = strings.TrimPrefix(path, "/v1/sandboxes/")
	if idx := strings.Index(path, "/"); idx != -1 {
		path = path[:idx]
	}
	return path
}

func extractTemplateID(path string) string {
	// Expected format: /v1/templates/<id>
	path = strings.TrimPrefix(path, "/v1/templates/")
	if idx := strings.Index(path, "/"); idx != -1 {
		path = path[:idx]
	}
	return path
}
