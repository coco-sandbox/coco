// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

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

// Sandbox represents a sandbox instance
type Sandbox struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	State     SandboxState      `json:"state"`
	Template  string            `json:"template"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	HostNode  string            `json:"host_node"`
	Config    map[string]any     `json:"config,omitempty"`
}

// AppState holds the application state
type AppState struct {
	sandboxes map[string]Sandbox
	mu        sync.RWMutex
	nodeID    string
	startTime time.Time
}

// HTTP Handlers

func healthHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(state.startTime).Seconds()
	resp := map[string]any{
		"healthy":        true,
		"version":        "0.1.0",
		"uptime_seconds": uptime,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func listSandboxesHandler(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	sandboxes := make([]Sandbox, 0, len(state.sandboxes))
	for _, sb := range state.sandboxes {
		sandboxes = append(sandboxes, sb)
	}

	resp := map[string]any{
		"sandboxes":    sandboxes,
		"total_count":  len(sandboxes),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func createSandboxHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name     string         `json:"name"`
		Template string         `json:"template"`
		Config   map[string]any `json:"config,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	id := fmt.Sprintf("sb_%s", uuid.New().String()[:8])
	now := time.Now()

	sb := Sandbox{
		ID:        id,
		Name:      req.Name,
		State:     SandboxStateRunning,
		Template:  req.Template,
		CreatedAt:  now,
		UpdatedAt:  now,
		HostNode:  state.nodeID,
		Config:    req.Config,
	}

	state.mu.Lock()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	log.Printf("Created sandbox %s (template: %s)", id, req.Template)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":      sb.ID,
		"name":    sb.Name,
		"state":   "running",
		"sandbox": sb,
	})
}

func describeSandboxHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/sandboxes/"):]

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("Sandbox %s not found", id), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sandbox": sb})
}

func destroySandboxHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/sandboxes/"):]

	state.mu.Lock()
	_, ok := state.sandboxes[id]
	if ok {
		delete(state.sandboxes, id)
	}
	state.mu.Unlock()

	if !ok {
		http.Error(w, fmt.Sprintf("Sandbox %s not found", id), http.StatusNotFound)
		return
	}

	log.Printf("Destroyed sandbox %s", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Sandbox %s destroyed", id),
	})
}

func execHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/sandboxes/"):len(r.URL.Path)-len("/exec")]

	state.mu.RLock()
	_, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("Sandbox %s not found", id), http.StatusNotFound)
		return
	}

	var req struct {
		Command []string `json:"cmd"`
		Env     map[string]string `json:"env,omitempty"`
		WorkingDir string `json:"working_dir,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Placeholder exec - in production would call cocovisor via Unix socket
	log.Printf("Exec in sandbox %s: %v", id, req.Command)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"stdout": "placeholder output",
		"exit_code": 0,
	})
}

var state *AppState

func main() {
	state = &AppState{
		sandboxes: make(map[string]Sandbox),
		nodeID:    uuid.New().String(),
		startTime: time.Now(),
	}

	// HTTP routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			listSandboxesHandler(w, r)
		} else if r.Method == http.MethodPost {
			createSandboxHandler(w, r)
		}
	})
	http.HandleFunc("/v1/sandboxes/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > len("/v1/sandboxes/") && path[len(path)-5:] == "/exec" {
			execHandler(w, r)
			return
		}
		if r.Method == http.MethodGet {
			describeSandboxHandler(w, r)
		} else if r.Method == http.MethodDelete {
			destroySandboxHandler(w, r)
		}
	})

	addr := ":4747"
	log.Printf("Coco core listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}