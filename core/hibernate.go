// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// =============================================================================
// Hibernate / Resume
// =============================================================================

// HibernateRequest represents a hibernate request
type HibernateRequest struct {
	Compress bool `json:"compress,omitempty"`
}

// HibernateResponse represents the hibernate response
type HibernateResponse struct {
	ID                string `json:"id"`
	State            string `json:"state"`
	HibernatePath    string `json:"hibernate_path"`
	SizeBytes        int64  `json:"size_bytes"`
	DurationMs       int64  `json:"duration_ms"`
}

// ResumeResponse represents the resume response
type ResumeResponse struct {
	ID          string `json:"id"`
	State      string `json:"state"`
	DurationMs int64  `json:"duration_ms"`
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

	var req HibernateRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Ignore parse errors, use defaults
		}
	}

	hibernatePath := fmt.Sprintf("/var/lib/coco/hibernation/%s", id)
	start := time.Now()

	// Create hibernation directory
	if err := os.MkdirAll(hibernatePath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create hibernation directory: %v", err))
		return
	}

	// TODO: Call cocovisor HIBERNATE frame when real implementation ready
	// For now, simulate with mock duration

	state.mu.Lock()
	sb.State = SandboxStateHibernated
	sb.HibernatePath = hibernatePath
	state.sandboxes[id] = sb
	state.mu.Unlock()

	duration := time.Since(start)
	sizeBytes := int64(sb.MemoryMB * 1024 * 1024) // Mock size

	state.metrics.RecordHibernate(duration, sizeBytes)
	log.Printf("Hibernated sandbox %s -> %s (size: %d bytes, duration: %v)", id, hibernatePath, sizeBytes, duration)

	// Audit log hibernate action
	auditLogger.LogSandboxAction(AuditActionHibernate, AuditResultSuccess, id, r, nil)

	writeJSON(w, http.StatusOK, HibernateResponse{
		ID:             id,
		State:         sb.State.String(),
		HibernatePath: hibernatePath,
		SizeBytes:     sizeBytes,
		DurationMs:    duration.Milliseconds(),
	})
}

func handleHibernateResume(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateHibernated {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s is not hibernated", id))
		return
	}

	start := time.Now()

	// TODO: Call cocovisor RESUME_HIBERNATED frame when real implementation ready
	// For now, simulate resume

	state.mu.Lock()
	sb.State = SandboxStateRunning
	sb.HibernatePath = ""
	state.sandboxes[id] = sb
	state.mu.Unlock()

	duration := time.Since(start)
	log.Printf("Resumed sandbox %s (duration: %v)", id, duration)

	// Audit log resume action
	auditLogger.LogSandboxAction(AuditActionResume, AuditResultSuccess, id, r, nil)

	writeJSON(w, http.StatusOK, ResumeResponse{
		ID:          id,
		State:      sb.State.String(),
		DurationMs: duration.Milliseconds(),
	})
}
