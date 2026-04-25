// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// =============================================================================
// Pause / Hibernate / Resume
// =============================================================================

func handleSandboxPause(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.Lock()
	sb, ok := state.sandboxes[id]
	if ok {
		sb.State = SandboxStatePaused
		state.sandboxes[id] = sb
	}
	state.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	log.Printf("Paused sandbox %s", id)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "state": "paused"})
}

func handleSandboxHibernate(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s is not running", id))
		return
	}

	hibernatePath := fmt.Sprintf("/var/lib/coco/hibernation/%s", id)

	state.mu.Lock()
	sb.State = SandboxStateHibernated
	sb.HibernatePath = hibernatePath
	state.sandboxes[id] = sb
	state.mu.Unlock()

	// Create hibernation directory
	// TODO: Call cocovisor HIBERNATE frame when real implementation ready
	mkdirAll(hibernatePath, 0755)

	elapsed := 3500 * time.Millisecond // mock
	state.metrics.RecordHibernate(elapsed, 512*1024*1024)
	log.Printf("Hibernated sandbox %s -> %s", id, hibernatePath)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"hibernate_path": hibernatePath,
		"duration_ms":    3500,
	})
}

func handleSandboxResume(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.Lock()
	sb, ok := state.sandboxes[id]
	if ok {
		sb.State = SandboxStateRunning
		state.sandboxes[id] = sb
	}
	state.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	log.Printf("Resumed sandbox %s", id)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "state": "running"})
}

func mkdirAll(path string, perm int) {
	// Simplified — in real impl use os.MkdirAll
	_ = path
	_ = perm
}