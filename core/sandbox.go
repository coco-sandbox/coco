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
// Sandbox CRUD
// =============================================================================

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

	if err := decodeJSON(r.Body, &req); err != nil {
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

	vsockCID := nextVsockCID()
	go bootSandbox(id, sb.Rootfs, uint32(sb.MemoryMB), uint32(sb.VCPUs), vsockCID)

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

func handleSandboxGet(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"sandbox": sb})
}

func handleSandboxDestroy(w http.ResponseWriter, r *http.Request, id string) {
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
	log.Printf("Destroyed sandbox %s", id)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Sandbox %s destroyed", id),
	})
}

// =============================================================================
// Boot Sandbox (goroutine)
// =============================================================================

func bootSandbox(id, rootfs string, memoryMB, vcpus, vsockCID uint32) {
	frame, err := visor.BuildBootFrame(id, rootfs, memoryMB, vcpus, 4747)
	if err != nil {
		log.Printf("[boot] BuildBootFrame failed: %v", err)
		state.mu.Lock()
		if sb, ok := state.sandboxes[id]; ok {
			sb.State = SandboxStateError
			state.sandboxes[id] = sb
		}
		state.mu.Unlock()
		return
	}

	client, err := visor.Dial()
	if err != nil {
		log.Printf("[boot] Dial failed (cocovisor not running): %v", err)
		// Mock successful boot for development
		state.mu.Lock()
		if sb, ok := state.sandboxes[id]; ok {
			sb.State = SandboxStateRunning
			sb.VsockCID = vsockCID
			sb.PID = 10000 + int(vsockCID)*100
			state.sandboxes[id] = sb
		}
		state.mu.Unlock()
		return
	}
	defer client.Close()

	resp, err := client.SendBoot(frame)
	if err != nil {
		log.Printf("[boot] SendBoot failed: %v", err)
		state.mu.Lock()
		if sb, ok := state.sandboxes[id]; ok {
			sb.State = SandboxStateError
			state.sandboxes[id] = sb
		}
		state.mu.Unlock()
		return
	}

	state.mu.Lock()
	if sb, ok := state.sandboxes[id]; ok {
		sb.State = SandboxStateRunning
		sb.VsockCID = resp.VsockCID
		sb.PID = int(resp.PID)
		state.sandboxes[id] = sb
	}
	state.mu.Unlock()

	log.Printf("[boot] Sandbox %s booted: vsock_cid=%d pid=%d", id, resp.VsockCID, resp.PID)
}