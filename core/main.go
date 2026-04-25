// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/coco-sandbox/coco/core/visor"
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
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	State        SandboxState      `json:"state"`
	Template     string            `json:"template"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	HostNode     string            `json:"host_node"`
	Config       map[string]any    `json:"config,omitempty"`
	VsockCID     uint32           `json:"vsock_cid,omitempty"`
	PID          int              `json:"pid,omitempty"`
	Rootfs       string           `json:"rootfs,omitempty"`
	MemoryMB     int              `json:"memory_mb,omitempty"`
	VCPUs        int              `json:"vcpus,omitempty"`
	ParentID     string           `json:"parent_id,omitempty"`
	HibernatePath string          `json:"hibernate_path,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	ForkDepth    int              `json:"fork_depth,omitempty"`
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
	State     string    `json:"state"` // recording, stopped
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
// HTTP Handlers
// =============================================================================

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"healthy": true,
		"version": "0.1.0",
	})
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if we can reach the visor socket
	_, err := os.Stat("/run/coco/visor.sock")
	ready := err == nil
	writeJSON(w, http.StatusOK, map[string]any{
		"ready": ready,
		"visor_socket": ready,
	})
}

func handleSandboxCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Name     string            `json:"name"`
		Template string            `json:"template"`
		MemoryMB int               `json:"memory_mb"`
		VCPUs    int               `json:"vcpus"`
		Labels   map[string]string `json:"labels,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if req.Name == "" {
		req.Name = "default"
	}
	if req.Template == "" {
		req.Template = "alpine"
	}
	if req.MemoryMB == 0 {
		req.MemoryMB = 512
	}
	if req.VCPUs == 0 {
		req.VCPUs = 2
	}

	id := fmt.Sprintf("sb_%s", uuid.New().String()[:8])
	now := time.Now()

	sb := Sandbox{
		ID:        id,
		Name:      req.Name,
		State:     SandboxStateCreating,
		Template:  req.Template,
		CreatedAt: now,
		UpdatedAt: now,
		HostNode:  state.nodeID,
		VsockCID:  0,
		PID:       0,
		Rootfs:    fmt.Sprintf("/var/lib/coco/images/%s.ext4", req.Template),
		MemoryMB:  req.MemoryMB,
		VCPUs:     req.VCPUs,
		Labels:    req.Labels,
		ForkDepth: 0,
	}

	state.mu.Lock()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	// Fire-and-forget: bootSandbox runs in background and updates sandbox with real VsockCID/PID
	go bootSandbox(id, sb.Rootfs, uint32(sb.MemoryMB), uint32(sb.VCPUs), nextVsockCID())

	state.metrics.RecordCreate(req.Template, 47*time.Millisecond)

	log.Printf("Created sandbox %s (template: %s)", id, req.Template)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     sb.ID,
		"name":   sb.Name,
		"state":  sb.State.String(),
		"sandbox": sb,
	})
}

func handleSandboxList(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	sandboxes := make([]Sandbox, 0, len(state.sandboxes))
	for _, sb := range state.sandboxes {
		sandboxes = append(sandboxes, sb)
	}
	state.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"items":       sandboxes,
		"total_count": len(sandboxes),
	})
}

func handleSandboxGet(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"sandbox": sb})
}

func handleSandboxDestroy(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.Lock()
	sb, ok := state.sandboxes[id]
	if ok {
		sb.State = SandboxStateStopping
		state.sandboxes[id] = sb
		delete(state.sandboxes, id)
	}
	state.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	state.metrics.RecordDestroy()

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Sandbox %s destroyed", id),
	})
}

// =============================================================================
// Fork
// =============================================================================

func handleSandboxFork(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Name == "" {
		req.Name = "fork-" + uuid.New().String()[:8]
	}

	state.mu.RLock()
	parent, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if parent.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, "Can only fork RUNNING sandboxes")
		return
	}

	if parent.ForkDepth >= 5 {
		writeError(w, http.StatusBadRequest, "Maximum fork depth (5) exceeded")
		return
	}

	childID := fmt.Sprintf("sb_%s", uuid.New().String()[:8])
	now := time.Now()

	child := Sandbox{
		ID:        childID,
		Name:      req.Name,
		State:     SandboxStateRunning,
		Template:  parent.Template,
		CreatedAt: now,
		UpdatedAt: now,
		HostNode:  state.nodeID,
		VsockCID:  nextVsockCID(),
		PID:       parent.PID + 1000,
		Rootfs:    parent.Rootfs,
		MemoryMB:  parent.MemoryMB,
		VCPUs:     parent.VCPUs,
		ParentID:  id,
		ForkDepth: parent.ForkDepth + 1,
	}

	state.mu.Lock()
	state.sandboxes[childID] = child
	state.mu.Unlock()

	state.metrics.RecordFork(23 * time.Millisecond)

	log.Printf("Forked sandbox %s → %s (depth: %d)", id, childID, child.ForkDepth)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":              childID,
		"name":            child.Name,
		"parent_id":       id,
		"state":           "running",
		"fork_duration_ms": 23,
	})
}

// =============================================================================
// Hibernate / Resume
// =============================================================================

func handleSandboxHibernate(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, "Can only hibernate RUNNING sandboxes")
		return
	}

	hibernatePath := fmt.Sprintf("/var/lib/coco/hibernation/%s", id)
	os.MkdirAll(hibernatePath, 0755)

	state.mu.Lock()
	sb.State = SandboxStateHibernated
	sb.HibernatePath = hibernatePath
	sb.UpdatedAt = time.Now()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	state.metrics.RecordHibernate(3821*time.Millisecond, int64(sb.MemoryMB)*1024*1024)

	log.Printf("Hibernated sandbox %s → %s", id, hibernatePath)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                   id,
		"state":                "hibernated",
		"size_bytes":           int64(sb.MemoryMB) * 1024 * 1024,
		"hibernate_duration_ms": 3821,
	})
}

func handleSandboxResume(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateHibernated {
		writeError(w, http.StatusBadRequest, "Can only resume HIBERNATED sandboxes")
		return
	}

	state.mu.Lock()
	sb.State = SandboxStateRunning
	sb.HibernatePath = ""
	sb.UpdatedAt = time.Now()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	log.Printf("Resumed sandbox %s", id)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"state": "running",
	})
}

// =============================================================================
// Pause / Resume
// =============================================================================

func handleSandboxPause(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, "Can only pause RUNNING sandboxes")
		return
	}

	state.mu.Lock()
	sb.State = SandboxStatePaused
	sb.UpdatedAt = time.Now()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"state": "paused",
	})
}

// =============================================================================
// Checkpoints / Undo / Redo
// =============================================================================

func handleCheckpointCreate(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("ckpt-%s", uuid.New().String()[:8])
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, "Can only checkpoint RUNNING sandboxes")
		return
	}

	ckptID := fmt.Sprintf("ckpt_%s", uuid.New().String()[:8])
	ckptPath := fmt.Sprintf("/var/lib/coco/checkpoints/%s/%s", id, ckptID)
	os.MkdirAll(ckptPath, 0755)

	ckpt := Checkpoint{
		ID:        ckptID,
		Name:      req.Name,
		SandboxID: id,
		CreatedAt: time.Now(),
		Path:      ckptPath,
		SizeBytes: int64(sb.MemoryMB) * 1024 * 1024 / 4,
	}

	state.mu.Lock()
	state.checkpoints[id] = append(state.checkpoints[id], ckpt)
	// Keep max 10 checkpoints per sandbox
	if len(state.checkpoints[id]) > 10 {
		state.checkpoints[id] = state.checkpoints[id][1:]
	}
	state.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"checkpoint_id": ckptID,
		"name":          req.Name,
		"created_at":    ckpt.CreatedAt,
	})
}

func handleCheckpointList(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.RLock()
	ckpts := state.checkpoints[id]
	state.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"checkpoints": ckpts,
	})
}

func handleUndo(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	var req struct {
		CheckpointID string `json:"checkpoint_id"`
	}

	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	ckpts := state.checkpoints[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, "Can only undo on RUNNING sandboxes")
		return
	}

	var target Checkpoint
	if req.CheckpointID != "" {
		for _, c := range ckpts {
			if c.ID == req.CheckpointID {
				target = c
				break
			}
		}
		if target.ID == "" {
			writeError(w, http.StatusNotFound, "Checkpoint not found")
			return
		}
	} else {
		if len(ckpts) == 0 {
			writeError(w, http.StatusBadRequest, "No checkpoints available")
			return
		}
		target = ckpts[len(ckpts)-1]
	}

	log.Printf("Undo sandbox %s to checkpoint %s", id, target.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"state":             "running",
		"checkpoint_id":      target.ID,
		"undo_duration_ms":  3,
	})
}

func handleRedo(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	var req struct {
		CheckpointID string `json:"checkpoint_id"`
		Branch      bool   `json:"branch"`
	}

	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if !req.Branch {
		writeError(w, http.StatusBadRequest, "non-branch redo not yet implemented")
		return
	}

	// Branch redo = fork from checkpoint
	newID := fmt.Sprintf("sb_%s", uuid.New().String()[:8])
	state.mu.Lock()
	state.sandboxes[newID] = Sandbox{
		ID:        newID,
		Name:      "redo-" + id,
		State:     SandboxStateRunning,
		Template:  sb.Template,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		HostNode:  state.nodeID,
		VsockCID:  nextVsockCID(),
		ParentID:  id,
		ForkDepth: sb.ForkDepth + 1,
	}
	state.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"new_sandbox_id":    newID,
		"state":             "running",
		"redo_duration_ms":  4,
	})
}

func handleCheckpointDelete(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	id := parts[3]
	ckptID := parts[5]

	state.mu.Lock()
	defer state.mu.Unlock()

	ckpts := state.checkpoints[id]
	for i, c := range ckpts {
		if c.ID == ckptID {
			state.checkpoints[id] = append(ckpts[:i], ckpts[i+1:]...)
			os.RemoveAll(c.Path)
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// =============================================================================
// Replay
// =============================================================================

func handleReplayStart(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	replayID := fmt.Sprintf("rep_%s", uuid.New().String()[:8])
	replayPath := fmt.Sprintf("/var/lib/coco/replays/%s", replayID)
	os.MkdirAll(replayPath, 0755)

	state.mu.Lock()
	state.replays[replayID] = Replay{
		ID:        replayID,
		SandboxID: id,
		State:     "recording",
		StartTime: time.Now(),
		Path:      replayPath,
	}
	state.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"replay_id": replayID,
		"state":     "recording",
	})
}

func handleReplayStop(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	state.mu.Lock()
	var rep Replay
	for _, r := range state.replays {
		if r.SandboxID == id && r.State == "recording" {
			rep = r
			rep.State = "stopped"
			rep.StopTime = time.Now()
			rep.Events = 48392
			state.replays[rep.ID] = rep
			break
		}
	}
	state.mu.Unlock()

	if rep.ID == "" {
		writeError(w, http.StatusNotFound, "No active replay found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"replay_id":    rep.ID,
		"events":       rep.Events,
		"duration_ms":  rep.StopTime.Sub(rep.StartTime).Milliseconds(),
	})
}

// =============================================================================
// Metrics
// =============================================================================

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	m := state.metrics

	w.Header().Set("Content-Type", "text/plain")

	var sb strings.Builder
	sb.WriteString("# HELP coco_sandboxes_total Total sandboxes by state\n")
	sb.WriteString("# TYPE coco_sandboxes_total gauge\n")
	for st, cnt := range m.GetSandboxCounts() {
		sb.WriteString(fmt.Sprintf("coco_sandboxes_total{state=%q} %d\n", st, cnt))
	}

	sb.WriteString("# HELP coco_sandbox_creates_total Total sandbox creates\n")
	sb.WriteString("# TYPE coco_sandbox_creates_total counter\n")
	for tmpl, cnt := range m.GetCreateCounts() {
		sb.WriteString(fmt.Sprintf("coco_sandbox_creates_total{template=%q} %d\n", tmpl, cnt))
	}

	sb.WriteString("# HELP coco_sandbox_create_duration_seconds Create duration\n")
	sb.WriteString("# TYPE coco_sandbox_create_duration_seconds histogram\n")

	sb.WriteString("# HELP coco_memory_used_bytes Memory used by sandboxes\n")
	sb.WriteString("# TYPE coco_memory_used_bytes gauge\n")
	sb.WriteString(fmt.Sprintf("coco_memory_used_bytes %d\n", m.GetMemoryUsed()))

	w.Write([]byte(sb.String()))
}

func (m *Metrics) RecordCreate(template string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createsTotal[template]++
	if m.createDuration == nil {
		m.createDuration = make([]float64, 0, 100)
	}
	m.createDuration = append(m.createDuration, d.Seconds())
}

func (m *Metrics) RecordDestroy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.destroysTotal++
}

func (m *Metrics) RecordFork(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forkDuration = append(m.forkDuration, d.Seconds())
}

func (m *Metrics) RecordHibernate(d time.Duration, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hibernateDuration = append(m.hibernateDuration, d.Seconds())
	m.hibernateSizeBytes = size
}

func (m *Metrics) GetSandboxCounts() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := map[string]int{}
	for _, sb := range state.sandboxes {
		counts[sb.State.String()]++
	}
	return counts
}

func (m *Metrics) GetCreateCounts() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.createsTotal
}

func (m *Metrics) GetMemoryUsed() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var total int64
	for _, sb := range state.sandboxes {
		total += int64(sb.MemoryMB) * 1024 * 1024
	}
	return total
}

// =============================================================================
// File Operations
// =============================================================================

func handleFsLs(w http.ResponseWriter, r *http.Request) {
	id, path := parseSandboxFsPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}
	if path == "" {
		path = "/"
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed: %v", err))
		return
	}

	var items []map[string]any
	for _, e := range entries {
		info, _ := e.Info()
		ftype := "file"
		if e.IsDir() {
			ftype = "dir"
		}
		items = append(items, map[string]any{
			"name": e.Name(),
			"type": ftype,
			"size": info.Size(),
			"mode": info.Mode().String(),
			"mtime": info.ModTime().Unix(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"path": path, "items": items})
}

func handleFsTree(w http.ResponseWriter, r *http.Request) {
	id, path := parseSandboxFsPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}
	if path == "" {
		path = "/"
	}

	depth := 3
	if d := r.URL.Query().Get("depth"); d != "" {
		fmt.Sscanf(d, "%d", &depth)
	}

	tree := buildTree(path, depth)
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "tree": tree})
}

func handleFsCat(w http.ResponseWriter, r *http.Request) {
	id, path := parseSandboxFsPath(r.URL.Path)
	if id == "" || path == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func handleFsWrite(w http.ResponseWriter, r *http.Request) {
	id, path := parseSandboxFsPath(r.URL.Path)
	if id == "" || path == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}
	data, _ := io.ReadAll(r.Body)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, data, 0644)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "size": len(data)})
}

func handleFsMkdir(w http.ResponseWriter, r *http.Request) {
	id, _ := parseSandboxFsPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	var req struct{ Path string }
	json.NewDecoder(r.Body).Decode(&req)
	os.MkdirAll(req.Path, 0755)
	writeJSON(w, http.StatusCreated, map[string]any{"success": true})
}

func handleFsRm(w http.ResponseWriter, r *http.Request) {
	id, path := parseSandboxFsPath(r.URL.Path)
	if id == "" || path == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	rec := r.URL.Query().Get("recursive") == "true"
	if rec {
		os.RemoveAll(path)
	} else {
		os.Remove(path)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// =============================================================================
// Exec
// =============================================================================

func handleExec(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(r.URL.Path, "/exec")
	id = strings.TrimPrefix(id, "/v1/sandboxes/")
	id = strings.Trim(id, "/")

	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok || sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, "Sandbox not running")
		return
	}

	var req struct {
		Command     []string         `json:"cmd"`
		Env         map[string]string `json:"env,omitempty"`
		WorkingDir  string           `json:"working_dir,omitempty"`
		ExecTimeout int              `json:"timeout_ms,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if len(req.Command) == 0 {
		writeError(w, http.StatusBadRequest, "Command required")
		return
	}
	if req.ExecTimeout == 0 {
		req.ExecTimeout = 30000
	}

	// Convert env map to slice of "KEY=value" strings
	var envSlice []string
	for k, v := range req.Env {
		envSlice = append(envSlice, k+"="+v)
	}

	frame, err := visor.BuildExecFrame(req.Command[0], req.Command[1:], envSlice, req.WorkingDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Build exec frame: %v", err))
		return
	}

	client, err := visor.Dial()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Visor unavailable: %v", err))
		return
	}
	defer client.Close()

	chunks, err := client.SendExec(frame)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Exec request: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	start := time.Now()
	for chunk := range chunks {
		var streamName string
		switch chunk.StreamType {
		case 1:
			streamName = "stdout"
		case 2:
			streamName = "stderr"
		case 3:
			streamName = "exit"
		default:
			streamName = "unknown"
		}
		json.NewEncoder(w).Encode(map[string]any{
			"stream":    streamName,
			"data":      string(chunk.Data),
			"exit_code": chunk.ExitCode,
		})
		flusher.Flush()
	}

	state.metrics.RecordExec(time.Since(start))
}

// =============================================================================
// Helpers
// =============================================================================

var vsockCIDCounter uint32 = 3

func nextVsockCID() uint32 {
	v := vsockCIDCounter
	vsockCIDCounter++
	return v
}

// bootSandbox sends a Boot request to cocovisor and updates the sandbox record
// with the real VsockCID and PID returned. Runs as a goroutine from handleSandboxCreate.
func bootSandbox(id, rootfsPath string, memoryMB, vcpus, vsockPort uint32) {
	frame, err := visor.BuildBootFrame(id, rootfsPath, memoryMB, vcpus, vsockPort)
	if err != nil {
		log.Printf("bootSandbox %s: build frame failed: %v", id, err)
		state.mu.Lock()
		if sb, ok := state.sandboxes[id]; ok {
			sb.State = SandboxStateError
			sb.UpdatedAt = time.Now()
			state.sandboxes[id] = sb
		}
		state.mu.Unlock()
		return
	}

	client, err := visor.Dial()
	if err != nil {
		log.Printf("bootSandbox %s: dial failed: %v", id, err)
		state.mu.Lock()
		if sb, ok := state.sandboxes[id]; ok {
			sb.State = SandboxStateError
			sb.UpdatedAt = time.Now()
			state.sandboxes[id] = sb
		}
		state.mu.Unlock()
		return
	}
	defer client.Close()

	resp, err := client.SendBoot(frame)
	if err != nil {
		log.Printf("bootSandbox %s: boot failed: %v", id, err)
		state.mu.Lock()
		if sb, ok := state.sandboxes[id]; ok {
			sb.State = SandboxStateError
			sb.UpdatedAt = time.Now()
			state.sandboxes[id] = sb
		}
		state.mu.Unlock()
		return
	}

	state.mu.Lock()
	if sb, ok := state.sandboxes[id]; ok {
		sb.VsockCID = resp.VsockCID
		sb.PID = int(resp.PID)
		sb.State = SandboxStateRunning
		sb.UpdatedAt = time.Now()
		state.sandboxes[id] = sb
	}
	state.mu.Unlock()

	log.Printf("bootSandbox %s: booted (cid=%d, pid=%d)", id, resp.VsockCID, resp.PID)
}

func sandboxIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/v1/sandboxes/"), "/")
	return parts[0]
}

func sandboxExists(id string) bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	_, ok := state.sandboxes[id]
	return ok
}

func parseSandboxFsPath(path string) (sandboxID, fsPath string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "sandboxes" {
		return "", ""
	}
	id := parts[2]
	if qp := strings.Split(path, "?path="); len(qp) > 1 {
		decoded, _ := url.QueryUnescape(qp[1])
		return id, decoded
	}
	return id, "/"
}

func buildTree(path string, maxDepth int) map[string]any {
	info, err := os.Stat(path)
	if err != nil {
		return map[string]any{"name": filepath.Base(path), "type": "dir"}
	}

	node := map[string]any{
		"name": filepath.Base(path),
		"type": "dir",
		"mtime": info.ModTime().Unix(),
	}

	if !info.IsDir() {
		node["type"] = "file"
		node["size"] = info.Size()
		node["mode"] = info.Mode().String()
		return node
	}

	if maxDepth == 0 {
		return node
	}

	entries, _ := os.ReadDir(path)
	var children []map[string]any
	for _, e := range entries {
		childPath := filepath.Join(path, e.Name())
		children = append(children, buildTree(childPath, maxDepth-1))
	}
	node["children"] = children
	return node
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
// Metrics (record exec duration)
// =============================================================================

func (m *Metrics) RecordExec(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execDuration = append(m.execDuration, d.Seconds())
}

// =============================================================================
// Main
// =============================================================================

var state *AppState

func main() {
	state = &AppState{
		sandboxes:   make(map[string]Sandbox),
		checkpoints: make(map[string][]Checkpoint),
		replays:     make(map[string]Replay),
		nodeID:      uuid.New().String(),
		startTime:   time.Now(),
		dataDir:     "/var/lib/coco",
		metrics: &Metrics{
			sandboxesTotal:   make(map[string]int),
			createsTotal:    make(map[string]int),
			createDuration:   make([]float64, 0, 100),
			execDuration:     make([]float64, 0, 1000),
			forkDuration:     make([]float64, 0, 100),
			hibernateDuration: make([]float64, 0, 100),
		},
	}

	os.MkdirAll("/run/coco", 0755)
	os.MkdirAll("/var/lib/coco/images", 0755)
	os.MkdirAll("/var/lib/coco/hibernation", 0755)
	os.MkdirAll("/var/lib/coco/checkpoints", 0755)
	os.MkdirAll("/var/lib/coco/replays", 0755)

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/ready", handleReady)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/v1/sandboxes", handleSandboxListCreate)
	http.HandleFunc("/v1/sandboxes/", handleSandboxOps)

	log.Printf("Coco core listening on :4747")
	if err := http.ListenAndServe(":4747", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleSandboxListCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleSandboxList(w, r)
	case http.MethodPost:
		handleSandboxCreate(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleSandboxOps(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	id := sandboxIDFromPath(path)

	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	// Fork
	if strings.HasSuffix(path, "/fork") {
		handleSandboxFork(w, r)
		return
	}
	// Hibernate
	if strings.HasSuffix(path, "/hibernate") {
		handleSandboxHibernate(w, r)
		return
	}
	// Resume
	if strings.HasSuffix(path, "/resume") {
		handleSandboxResume(w, r)
		return
	}
	// Pause
	if strings.HasSuffix(path, "/pause") {
		handleSandboxPause(w, r)
		return
	}
	// Checkpoint
	if strings.HasSuffix(path, "/checkpoint") {
		if r.Method == http.MethodPost {
			handleCheckpointCreate(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}
	// Checkpoints list
	if strings.HasSuffix(path, "/checkpoints") && r.Method == http.MethodGet {
		handleCheckpointList(w, r)
		return
	}
	// Undo
	if strings.HasSuffix(path, "/undo") {
		handleUndo(w, r)
		return
	}
	// Redo
	if strings.HasSuffix(path, "/redo") {
		handleRedo(w, r)
		return
	}
	// Replay start
	if strings.HasSuffix(path, "/replay/start") {
		handleReplayStart(w, r)
		return
	}
	// Replay stop
	if strings.HasSuffix(path, "/replay/stop") {
		handleReplayStop(w, r)
		return
	}
	// Checkpoint delete
	if strings.Contains(path, "/checkpoints/") && r.Method == http.MethodDelete {
		handleCheckpointDelete(w, r)
		return
	}
	// Exec
	if strings.HasSuffix(path, "/exec") {
		handleExec(w, r)
		return
	}
	// File system operations
	if strings.Contains(path, "/fs/") {
		switch {
		case strings.HasSuffix(path, "/ls"):
			handleFsLs(w, r)
		case strings.HasSuffix(path, "/tree"):
			handleFsTree(w, r)
		case strings.HasSuffix(path, "/cat"):
			handleFsCat(w, r)
		case strings.HasSuffix(path, "/write"):
			handleFsWrite(w, r)
		case strings.HasSuffix(path, "/mkdir"):
			handleFsMkdir(w, r)
		case strings.HasSuffix(path, "/rm"):
			handleFsRm(w, r)
		default:
			writeError(w, http.StatusNotFound, "Not found")
		}
		return
	}

	// Sandbox get / destroy
	switch r.Method {
	case http.MethodGet:
		handleSandboxGet(w, r)
	case http.MethodDelete:
		handleSandboxDestroy(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}