// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Command coco-core is the main API server for the Coco sandbox runtime.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/internal/config"
	"github.com/coco-sandbox/coco/internal/metrics"
	"github.com/coco-sandbox/coco/internal/template"
	"github.com/coco-sandbox/coco/internal/types"
	"github.com/coco-sandbox/coco/pkg/api/handlers"
	"github.com/coco-sandbox/coco/pkg/cluster"
	prommetrics "github.com/coco-sandbox/coco/pkg/metrics"
	"github.com/coco-sandbox/coco/pkg/middleware"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/visor"
)

type server struct {
	config      *config.Config
	mux         *http.ServeMux
	server      *http.Server
	store       *store.BadgerStore
	cluster     *cluster.Manager
	metrics     *metrics.Metrics
	templateMgr *template.Manager
	visorPool   *visor.Pool
	wg          sync.WaitGroup
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

	// Initialize BadgerDB store
	st, err := store.NewBadgerStore(s.config.StoreDir)
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}
	s.store = st

	// Initialize visor client pool
	s.visorPool = visor.NewPool(visor.SocketPath, 10)

	// Initialize cluster manager
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Warning: failed to get hostname: %v", err)
	}
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

	// Start Prometheus metrics server on port 9090
	go func() {
		prommetrics.Register()
		http.Handle("/metrics", prommetrics.Handler())
		log.Printf("Metrics server listening on :9090")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

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
	s.mux.HandleFunc("/v1/sandboxes", handleSandboxes(s, sandboxHandler))
	s.mux.HandleFunc("/v1/sandboxes/", handleSandboxByID(s, sandboxHandler, execHandler))

	// Templates
	s.mux.HandleFunc("/v1/templates", handleTemplates(s, templateHandler))
	s.mux.HandleFunc("/v1/templates/", handleTemplateByID(s, templateHandler))

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

	s.wg.Wait()
	s.visorPool.Close()
	s.cluster.Stop()
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

func handleSandboxes(s *server, h *handlers.SandboxHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.wg.Add(1)
		defer s.wg.Done()
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

func handleSandboxByID(s *server, sandboxHandler *handlers.SandboxHandler, execHandler *handlers.ExecHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.wg.Add(1)
		defer s.wg.Done()
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

func handleTemplates(s *server, h *handlers.TemplateHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.wg.Add(1)
		defer s.wg.Done()
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

func handleTemplateByID(s *server, h *handlers.TemplateHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.wg.Add(1)
		defer s.wg.Done()
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

// sandboxService implements handlers.SandboxService
type sandboxService struct {
	store      *store.BadgerStore
	metrics    *metrics.Metrics
	visorPool  *visor.Pool
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
	sb, err := s.store.GetSandbox(id)
	if err != nil {
		return fmt.Errorf("sandbox %s not found: %w", id, err)
	}

	visorConn, err := s.visorPool.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire visor connection: %w", err)
	}
	defer s.visorPool.Release(visorConn)

	if err := visorConn.Pause(id); err != nil {
		return fmt.Errorf("visor pause failed: %w", err)
	}

	sb.State = types.SandboxStatePaused
	if err := s.store.PutSandbox(sb); err != nil {
		return fmt.Errorf("failed to update sandbox state: %w", err)
	}

	return nil
}

func (s *sandboxService) Resume(ctx context.Context, id string) error {
	sb, err := s.store.GetSandbox(id)
	if err != nil {
		return fmt.Errorf("sandbox %s not found: %w", id, err)
	}

	visorConn, err := s.visorPool.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire visor connection: %w", err)
	}
	defer s.visorPool.Release(visorConn)

	if err := visorConn.Resume(id); err != nil {
		return fmt.Errorf("visor resume failed: %w", err)
	}

	sb.State = types.SandboxStateRunning
	if err := s.store.PutSandbox(sb); err != nil {
		return fmt.Errorf("failed to update sandbox state: %w", err)
	}

	return nil
}

func (s *sandboxService) Fork(ctx context.Context, parentID string, req *types.ForkRequest) (*types.Sandbox, error) {
	parent, err := s.store.GetSandbox(parentID)
	if err != nil {
		return nil, fmt.Errorf("parent sandbox %s not found: %w", parentID, err)
	}

	visorConn, err := s.visorPool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire visor connection: %w", err)
	}
	defer s.visorPool.Release(visorConn)

	forkResp, err := visorConn.Fork(parentID)
	if err != nil {
		return nil, fmt.Errorf("visor fork failed: %w", err)
	}

	childID := generateID()
	child := &types.Sandbox{
		ID:        childID,
		Name:      req.Name,
		State:     types.SandboxStateRunning,
		ParentID:  parentID,
		Template:  parent.Template,
		MemoryMB:  parent.MemoryMB,
		VCPUs:     parent.VCPUs,
		Labels:    parent.Labels,
		Created:   time.Now(),
	}

	if req.Labels != nil {
		for k, v := range req.Labels {
			child.Labels[k] = v
		}
	}

	if err := s.store.PutSandbox(child); err != nil {
		return nil, fmt.Errorf("failed to store child sandbox: %w", err)
	}

	s.metrics.RecordFork(int64(forkResp.DurationMs))

	return child, nil
}

func (s *sandboxService) Hibernate(ctx context.Context, id string) error {
	sb, err := s.store.GetSandbox(id)
	if err != nil {
		return fmt.Errorf("sandbox %s not found: %w", id, err)
	}

	visorConn, err := s.visorPool.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire visor connection: %w", err)
	}
	defer s.visorPool.Release(visorConn)

	if err := visorConn.Hibernate(id); err != nil {
		return fmt.Errorf("visor hibernate failed: %w", err)
	}

	sb.State = types.SandboxStateHibernated
	if err := s.store.PutSandbox(sb); err != nil {
		return fmt.Errorf("failed to update sandbox state: %w", err)
	}

	return nil
}

func (s *sandboxService) ResumeHibernate(ctx context.Context, id string) error {
	sb, err := s.store.GetSandbox(id)
	if err != nil {
		return fmt.Errorf("sandbox %s not found: %w", id, err)
	}

	visorConn, err := s.visorPool.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire visor connection: %w", err)
	}
	defer s.visorPool.Release(visorConn)

	bootResp, err := visorConn.ResumeHibernated(id)
	if err != nil {
		return fmt.Errorf("visor resume-hibernate failed: %w", err)
	}

	sb.State = types.SandboxStateRunning
	sb.VsockCID = bootResp.VsockCID
	if err := s.store.PutSandbox(sb); err != nil {
		return fmt.Errorf("failed to update sandbox state: %w", err)
	}

	return nil
}

func generateID() string {
	return "sb_" + time.Now().Format("20060102150405")
}

var _ handlers.SandboxService = (*sandboxService)(nil)
var _ handlers.TemplateService = (*template.Manager)(nil)

// ID extraction helpers

func extractSandboxID(path string) string {
	const prefix = "/v1/sandboxes/"
	idx := strings.Index(path, prefix)
	if idx == -1 {
		return ""
	}
	remaining := path[idx+len(prefix):]
	// Remove any trailing path segments
	if i := strings.Index(remaining, "/"); i != -1 {
		remaining = remaining[:i]
	}
	// Validate non-empty
	if remaining == "" {
		return ""
	}
	return remaining
}

func extractTemplateID(path string) string {
	// Expected format: /v1/templates/<id>
	path = strings.TrimPrefix(path, "/v1/templates/")
	if idx := strings.Index(path, "/"); idx != -1 {
		path = path[:idx]
	}
	return path
}
