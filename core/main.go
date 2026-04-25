// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coco-sandbox/coco/core/config"
)

// =============================================================================
// Types
// =============================================================================

type SandboxState int

const (
	SandboxStateCreating SandboxState = iota
	SandboxStateRunning
	SandboxStatePaused
	SandboxStateHibernated
	SandboxStateStopping
	SandboxStateStopped
	SandboxStateError
)

func (s SandboxState) String() string {
	switch s {
	case SandboxStateCreating:
		return "creating"
	case SandboxStateRunning:
		return "running"
	case SandboxStatePaused:
		return "paused"
	case SandboxStateHibernated:
		return "hibernated"
	case SandboxStateStopping:
		return "stopping"
	case SandboxStateStopped:
		return "stopped"
	case SandboxStateError:
		return "error"
	default:
		return "unknown"
	}
}

type Sandbox struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	State         SandboxState   `json:"state"`
	Template      string         `json:"template"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	HostNode      string         `json:"host_node"`
	Config        map[string]any `json:"config,omitempty"`
	VsockCID      uint32         `json:"vsock_cid,omitempty"`
	PID           int            `json:"pid,omitempty"`
	Rootfs        string         `json:"rootfs,omitempty"`
	MemoryMB      int            `json:"memory_mb,omitempty"`
	VCPUs         int            `json:"vcpus,omitempty"`
	ParentID      string         `json:"parent_id,omitempty"`
	HibernatePath string         `json:"hibernate_path,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ForkDepth     int            `json:"fork_depth,omitempty"`
}

type Checkpoint struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SandboxID string    `json:"sandbox_id"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
}

type Replay struct {
	ID        string    `json:"id"`
	SandboxID string    `json:"sandbox_id"`
	State     string    `json:"state"`
	Events    int       `json:"events,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	StopTime  time.Time `json:"stop_time,omitempty"`
	Path      string    `json:"path,omitempty"`
}

type AppState struct {
	sandboxes   map[string]Sandbox
	checkpoints map[string][]Checkpoint
	replays     map[string]Replay
	mu          sync.RWMutex
	nodeID      string
	startTime   time.Time
	dataDir     string
	metrics     *Metrics
}

type Metrics struct {
	mu                    sync.RWMutex
	sandboxesTotal       map[string]int
	createsTotal         map[string]int
	createDuration       []float64
	destroysTotal        int
	execDuration         []float64
	forkDuration         []float64
	hibernateDuration    []float64
	hibernateSizeBytes   int64
	memoryUsedBytes      int64
	cpuSecondsTotal      float64
	networkBytesIngress  int64
	networkBytesEgress   int64
}

// =============================================================================
// Global State
// =============================================================================

var state AppState
var auditLogger *AuditLogger
var vsockMu sync.Mutex
var nextVsockCID uint32 = 3

func allocateVsockCID() uint32 {
	vsockMu.Lock()
	c := nextVsockCID
	nextVsockCID++
	vsockMu.Unlock()
	return c
}

// =============================================================================
// Shared Helpers
// =============================================================================

func sandboxIDFromPath(path string) string {
	parts := strings.Split(path, "/v1/sandboxes/")
	if len(parts) < 2 {
		return ""
	}
	idPart := strings.SplitN(parts[1], "/", 2)[0]
	idPart = strings.SplitN(idPart, "?", 2)[0]
	return idPart
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

// =============================================================================
// Main
// =============================================================================

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	// Load configuration
	cfg := config.Default()

	state = AppState{
		sandboxes:   make(map[string]Sandbox),
		checkpoints: make(map[string][]Checkpoint),
		replays:     make(map[string]Replay),
		nodeID:      getNodeID(),
		startTime:   time.Now(),
		dataDir:     cfg.DataDir,
		metrics:     newMetrics(),
	}

	// Initialize audit logger
	auditLogger = NewAuditLogger(log.Default())

	// Initialize cluster manager
	clusterManager = NewClusterManager(state.nodeID, "0.1.0")
	clusterManager.Start()

	// Initialize Raft consensus
	raft = NewRaftConsensus(state.nodeID, clusterManager)
	raft.Start()

	// Initialize vsock router for cross-node communication
	vsockRouter = NewVsockRouter(allocateVsockCID(), state.nodeID)
	if err := vsockRouter.Start(); err != nil {
		log.Printf("[vsock-router] Failed to start: %v (continuing anyway)", err)
	}

	setupDirectories()
	mux := http.NewServeMux()
	setupRoutes(mux)

	// Wrap with middleware chain
	handler := wrapMiddleware(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start server in goroutine
	go func() {
		log.Printf("coco-core starting on %s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	// Shutdown cluster manager
	clusterManager.Stop()

	// Shutdown Raft consensus
	raft.Stop()

	// Shutdown vsock router
	vsockRouter.Stop()

	log.Println("coco-core stopped")
}

func setupDirectories() {
	dirs := []string{
		"/run/coco",
		"/var/lib/coco/images",
		"/var/lib/coco/hibernation",
		"/var/lib/coco/checkpoints",
		"/var/lib/coco/replays",
		"/var/lib/coco/store",
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}
}

func getNodeID() string {
	hostname, _ := os.Hostname()
	return hostname
}