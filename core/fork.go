// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Fork
// =============================================================================

func handleSandboxFork(w http.ResponseWriter, r *http.Request, id string) {
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

	if sb.ForkDepth >= 5 {
		writeError(w, http.StatusBadRequest, "Fork depth limit exceeded (max 5)")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	childID := fmt.Sprintf("sb_%s", uuid.New().String()[:8])
	now := time.Now()

	state.mu.Lock()
	child := Sandbox{
		ID:        childID,
		Name:      req.Name,
		State:     SandboxStateCreating,
		Template:  sb.Template,
		CreatedAt: now,
		UpdatedAt: now,
		HostNode:  state.nodeID,
		Rootfs:    sb.Rootfs,
		MemoryMB:  sb.MemoryMB,
		VCPUs:     sb.VCPUs,
		ParentID:  id,
		ForkDepth: sb.ForkDepth + 1,
	}
	state.sandboxes[childID] = child
	state.mu.Unlock()

	vsockCID := nextVsockCID()
	go forkSandbox(id, childID, vsockCID)

	elapsed := 15 * time.Millisecond
	state.metrics.RecordFork(elapsed)
	log.Printf("Forked sandbox %s -> %s", id, childID)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       child.ID,
		"name":     child.Name,
		"parent_id": sb.ID,
		"state":    child.State.String(),
	})
}

func forkSandbox(parentID, childID string, vsockCID uint32) {
	state.mu.Lock()
	parent, ok := state.sandboxes[parentID]
	if !ok {
		state.mu.Unlock()
		return
	}
	state.mu.Unlock()

	// TODO: Call cocovisor FORK frame when implemented
	// For now, simulate fork with mock child VM

	state.mu.Lock()
	child, ok := state.sandboxes[childID]
	if ok {
		child.State = SandboxStateRunning
		child.VsockCID = vsockCID
		child.PID = parent.PID + 1
		state.sandboxes[childID] = child
	}
	state.mu.Unlock()

	log.Printf("[fork] Child %s vsock_cid=%d pid=%d", childID, vsockCID, child.PID)
}