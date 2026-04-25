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
)

// =============================================================================
// Types
// =============================================================================

// SandboxState represents the state of a sandbox
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

// Sandbox represents a sandbox instance
type Sandbox struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	State     SandboxState      `json:"state"`
	Template  string            `json:"template"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	HostNode  string            `json:"host_node"`
	Config    map[string]any    `json:"config,omitempty"`
	VsockCID  uint32            `json:"vsock_cid,omitempty"`
	PID       int               `json:"pid,omitempty"`
	Rootfs    string            `json:"rootfs,omitempty"`
}

// FileEntry represents a file or directory entry
type FileEntry struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "file" or "dir"
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	MTime   int64  `json:"mtime"`
}

// TreeNode represents a node in the directory tree
type TreeNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Size     int64       `json:"size,omitempty"`
	Mode     string      `json:"mode,omitempty"`
	MTime    int64       `json:"mtime,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}

// AppState holds the application state
type AppState struct {
	sandboxes map[string]Sandbox
	mu        sync.RWMutex
	nodeID    string
	startTime time.Time
	visorsock string
}

// =============================================================================
// HTTP Handlers
// =============================================================================

// Health check
func handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(state.startTime).Seconds()
	resp := map[string]any{
		"healthy":        true,
		"version":        "0.1.0",
		"uptime_seconds": uptime,
		"node_id":        state.nodeID,
		"sandboxes":      len(state.sandboxes),
	}
	writeJSON(w, http.StatusOK, resp)
}

// List sandboxes
func handleListSandboxes(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	sandboxes := make([]Sandbox, 0, len(state.sandboxes))
	for _, sb := range state.sandboxes {
		sandboxes = append(sandboxes, sb)
	}
	state.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      sandboxes,
		"total_count": len(sandboxes),
	})
}

// Create sandbox
func handleCreateSandbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Name     string         `json:"name"`
		Template string         `json:"template"`
		MemoryMB int            `json:"memory_mb"`
		VCPUs    int            `json:"vcpus"`
		Config   map[string]any `json:"config,omitempty"`
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
		State:     SandboxStateRunning,
		Template:  req.Template,
		CreatedAt: now,
		UpdatedAt: now,
		HostNode:  state.nodeID,
		Config:    req.Config,
		VsockCID:  3, // starting from 3, assign sequentially
		PID:       0, // placeholder
		Rootfs:    fmt.Sprintf("/var/lib/coco/images/%s.rootfs", req.Template),
	}

	state.mu.Lock()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	log.Printf("Created sandbox %s (template: %s, mem: %dMB, vcpus: %d)", id, req.Template, req.MemoryMB, req.VCPUs)

	// Call cocovisor to boot
	go bootSandbox(id, req.Template, req.MemoryMB, req.VCPUs)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     sb.ID,
		"name":   sb.Name,
		"state":  sb.State.String(),
		"sandbox": sb,
	})
}

// Get sandbox details
func handleGetSandbox(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/sandboxes/")
	id = strings.TrimSuffix(id, "/")

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"sandbox": sb})
}

// Delete sandbox
func handleDeleteSandbox(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/sandboxes/")
	id = strings.TrimSuffix(id, "/")

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

	log.Printf("Destroyed sandbox %s", id)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Sandbox %s destroyed", id),
	})
}

// Exec command in sandbox (streaming)
func handleExec(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(r.URL.Path, "/exec")
	id = strings.TrimSuffix(id, "/v1/sandboxes/")
	id = strings.Trim(strings.TrimPrefix(id, "/v1/sandboxes/"), "/exec")

	state.mu.RLock()
	_, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
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
		writeError(w, http.StatusBadRequest, "Command is required")
		return
	}

	if req.ExecTimeout == 0 {
		req.ExecTimeout = 30000 // 30s default
	}

	// For streaming response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	// Simulate exec (real implementation would use visor socket)
	cmdline := strings.Join(req.Command, " ")
	log.Printf("Exec in sandbox %s: %s", id, cmdline)

	// Write streaming response
	resp := map[string]any{
		"stream": "stdout",
		"data":   fmt.Sprintf("$ %s\nHello from sandbox %s!\nCommand executed successfully.\n", cmdline, id),
	}
	json.NewEncoder(w).Encode(resp)
	w.(http.Flusher).Flush()

	// Send exit
	exitResp := map[string]any{
		"stream":   "exit",
		"exit_code": 0,
	}
	json.NewEncoder(w).Encode(exitResp)
}

// =============================================================================
// File Operations (fs endpoints)
// =============================================================================

// GET /v1/sandboxes/:id/fs/ls?path=...
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

	entries, err := listDirectory(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list directory: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path": path,
		"items": entries,
	})
}

// GET /v1/sandboxes/:id/fs/tree?path=...&depth=...
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

	depth := 0
	if d := r.URL.Query().Get("depth"); d != "" {
		fmt.Sscanf(d, "%d", &depth)
	}

	tree, err := buildTree(path, depth)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to build tree: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path": path,
		"tree": tree,
	})
}

// GET /v1/sandboxes/:id/fs/cat?path=...
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

	content, err := readFile(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(content)
}

// PUT /v1/sandboxes/:id/fs/write?path=...
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read body: %v", err))
		return
	}

	if err := writeFile(path, body); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to write file: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"path":    path,
		"size":    len(body),
	})
}

// POST /v1/sandboxes/:id/fs/mkdir
func handleFsMkdir(w http.ResponseWriter, r *http.Request) {
	id, _ := parseSandboxFsPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "Path is required")
		return
	}

	if err := os.MkdirAll(req.Path, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"path":    req.Path,
	})
}

// DELETE /v1/sandboxes/:id/fs/rm?path=...&recursive=...
func handleFsRm(w http.ResponseWriter, r *http.Request) {
	id, path := parseSandboxFsPath(r.URL.Path)
	if id == "" || path == "" {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if !sandboxExists(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	recursive := r.URL.Query().Get("recursive") == "true"

	if recursive {
		if err := os.RemoveAll(path); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove: %v", err))
			return
		}
	} else {
		if err := os.Remove(path); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to remove: %v", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"path":    path,
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func parseSandboxFsPath(path string) (sandboxID, fsPath string) {
	// Format: /v1/sandboxes/{id}/fs/{op}
	// or: /v1/sandboxes/{id}/fs/{op}?path=...
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "sandboxes" {
		return "", ""
	}
	id := parts[2]
	fsIndex := 3
	if parts[3] == "" {
		fsIndex = 4
	}

	// Extract query path if present
	if qp := strings.Split(path, "?path="); len(qp) > 1 {
		decoded, _ := url.QueryUnescape(qp[1])
		return id, decoded
	}

	return id, "/"
}

func sandboxExists(id string) bool {
	state.mu.RLock()
	_, ok := state.sandboxes[id]
	state.mu.RUnlock()
	return ok
}

func listDirectory(path string) ([]FileEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []FileEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		ftype := "file"
		if entry.IsDir() {
			ftype = "dir"
		}

		result = append(result, FileEntry{
			Name:  entry.Name(),
			Type:  ftype,
			Size:  info.Size(),
			Mode:  info.Mode().String(),
			MTime: info.ModTime().Unix(),
		})
	}

	return result, nil
}

func buildTree(path string, maxDepth int) (*TreeNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &TreeNode{
		Name:  filepath.Base(path),
		Type:  "dir",
		MTime: info.ModTime().Unix(),
	}

	if !info.IsDir() {
		node.Type = "file"
		node.Size = info.Size()
		node.Mode = info.Mode().String()
		return node, nil
	}

	if maxDepth != 0 {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil
		}

		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			child, err := buildTree(childPath, maxDepth-1)
			if err != nil {
				continue
			}
			node.Children = append(node.Children, child)
		}
	}

	return node, nil
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func bootSandbox(id, template string, memoryMB, vcpus int) {
	log.Printf("Booting sandbox %s via cocovisor (template=%s, mem=%dMB, vcpus=%d)", id, template, memoryMB, vcpus)
	// Placeholder: real implementation calls cocovisor Unix socket
}

// =============================================================================
// Main
// =============================================================================

var state *AppState

func main() {
	state = &AppState{
		sandboxes: make(map[string]Sandbox),
		nodeID:    uuid.New().String(),
		startTime: time.Now(),
		visorsock: "/run/coco/visor.sock",
	}

	// Ensure socket dir exists
	os.MkdirAll("/run/coco", 0755)
	os.MkdirAll("/var/lib/coco/images", 0755)

	// Register routes
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/v1/sandboxes", handleRouter)
	http.HandleFunc("/v1/sandboxes/", handleSandboxRouter)

	addr := ":4747"
	log.Printf("Coco core listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListSandboxes(w, r)
	case http.MethodPost:
		handleCreateSandbox(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleSandboxRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// File system operations: /v1/sandboxes/{id}/fs/{op}
	if strings.Contains(path, "/fs/") {
		if strings.HasSuffix(path, "/ls") {
			handleFsLs(w, r)
			return
		}
		if strings.HasSuffix(path, "/tree") {
			handleFsTree(w, r)
			return
		}
		if strings.HasSuffix(path, "/cat") {
			handleFsCat(w, r)
			return
		}
		if strings.HasSuffix(path, "/write") {
			handleFsWrite(w, r)
			return
		}
		if strings.HasSuffix(path, "/mkdir") {
			handleFsMkdir(w, r)
			return
		}
		if strings.HasSuffix(path, "/rm") {
			handleFsRm(w, r)
			return
		}
	}

	// Sandbox operations
	if strings.HasSuffix(path, "/exec") {
		handleExec(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetSandbox(w, r)
	case http.MethodDelete:
		handleDeleteSandbox(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}