// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Checkpoint
// =============================================================================

func handleCheckpointCreate(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if req.Name == "" {
		req.Name = "checkpoint"
	}

	cp := Checkpoint{
		ID:        fmt.Sprintf("cp_%s", uuid.New().String()[:8]),
		Name:      req.Name,
		SandboxID: id,
		CreatedAt: time.Now(),
		Path:      fmt.Sprintf("/var/lib/coco/checkpoints/%s/%s", id, uuid.New().String()[:8]),
		SizeBytes: 64 * 1024 * 1024, // mock 64MB
	}

	state.mu.Lock()
	state.checkpoints[id] = append(state.checkpoints[id], cp)
	state.mu.Unlock()

	mkdirAll(filepath.Dir(cp.Path), 0755)

	log.Printf("Created checkpoint %s for sandbox %s", cp.ID, id)
	writeJSON(w, http.StatusCreated, map[string]any{"checkpoint": cp})
}

func handleCheckpointList(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	cps, ok := state.checkpoints[id]
	state.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"items": []Checkpoint{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": cps})
}

func handleUndo(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		CheckpointID string `json:"checkpoint_id"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// TODO: real undo using checkpoint diff
	log.Printf("Undo to checkpoint %s for sandbox %s", req.CheckpointID, id)
	elapsed := 3 * time.Millisecond // mock

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"duration_ms": elapsed,
	})
}

func handleRedo(w http.ResponseWriter, r *http.Request, id string) {
	// TODO: real redo
	log.Printf("Redo for sandbox %s", id)
	elapsed := 4 * time.Millisecond // mock

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"duration_ms": elapsed,
	})
}