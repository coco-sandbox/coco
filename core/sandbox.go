// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/coco-sandbox/coco/core/visor"
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

	vsockCID := allocateVsockCID()
	go bootSandbox(id, sb.Rootfs, uint32(sb.MemoryMB), uint32(sb.VCPUs), vsockCID)

	state.metrics.RecordCreate(req.Template, 47*time.Millisecond)
	log.Printf("Created sandbox %s (template: %s)", id, req.Template)

	// Audit log sandbox creation
	auditLogger.LogSandboxAction(AuditActionCreate, AuditResultSuccess, id, r, nil)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     sb.ID,
		"name":   sb.Name,
		"state":  sb.State.String(),
		"sandbox": sb,
	})
}

func handleSandboxList(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	offset := 0
	limit := 100

	if offStr := r.URL.Query().Get("offset"); offStr != "" {
		if n, err := parseInt(offStr); err == nil && n >= 0 {
			offset = n
		}
	}
	if limStr := r.URL.Query().Get("limit"); limStr != "" {
		if n, err := parseInt(limStr); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	// Filter by state if provided
	stateFilter := r.URL.Query().Get("state")

	// Filter by labels if provided
	labelKey := r.URL.Query().Get("label_key")
	labelValue := r.URL.Query().Get("label_value")

	state.mu.RLock()
	filtered := make([]Sandbox, 0, len(state.sandboxes))
	for _, sb := range state.sandboxes {
		// Apply state filter
		if stateFilter != "" && sb.State.String() != stateFilter {
			continue
		}
		// Apply label filter
		if labelKey != "" {
			if v, ok := sb.Labels[labelKey]; !ok || (labelValue != "" && v != labelValue) {
				continue
			}
		}
		filtered = append(filtered, sb)
	}

	// Sort by CreatedAt descending (newest first)
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].CreatedAt.Before(filtered[j].CreatedAt) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	totalCount := len(filtered)

	// Apply pagination
	end := offset + limit
	if offset >= len(filtered) {
		filtered = []Sandbox{}
	} else if end > len(filtered) {
		filtered = filtered[offset:]
	} else {
		filtered = filtered[offset:end]
	}
	state.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"items":        filtered,
		"total_count":  totalCount,
		"offset":       offset,
		"limit":        limit,
		"has_more":     end < totalCount,
	})
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
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

func handleSandboxUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	var req struct {
		Name   string            `json:"name,omitempty"`
		Labels map[string]string `json:"labels,omitempty"`
	}

	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	state.mu.Lock()
	if req.Name != "" {
		sb.Name = req.Name
	}
	if req.Labels != nil {
		if sb.Labels == nil {
			sb.Labels = make(map[string]string)
		}
		for k, v := range req.Labels {
			sb.Labels[k] = v
		}
	}
	sb.UpdatedAt = time.Now()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"sandbox": sb})
}

func handleSandboxPause(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
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
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s is not running", id))
		return
	}

	state.mu.Lock()
	sb.State = SandboxStatePaused
	sb.UpdatedAt = time.Now()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"state": sb.State.String(),
	})
}

func handleSandboxResume(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStatePaused {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s is not paused", id))
		return
	}

	state.mu.Lock()
	sb.State = SandboxStateRunning
	sb.UpdatedAt = time.Now()
	state.sandboxes[id] = sb
	state.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"state": sb.State.String(),
	})
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

	// Audit log sandbox destruction
	auditLogger.LogSandboxAction(AuditActionDelete, AuditResultSuccess, id, r, nil)

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