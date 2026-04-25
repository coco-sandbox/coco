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

	"github.com/google/uuid"
)

// =============================================================================
// Checkpoint Request/Response Types
// =============================================================================

type CreateCheckpointRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type CreateCheckpointResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SandboxID   string    `json:"sandbox_id"`
	CreatedAt   time.Time `json:"created_at"`
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"size_bytes"`
	DurationMs  int64     `json:"duration_ms"`
	ParentID    string    `json:"parent_id,omitempty"`
	Description string    `json:"description,omitempty"`
}

type ListCheckpointsResponse struct {
	Items      []Checkpoint `json:"items"`
	TotalCount int           `json:"total_count"`
}

type UndoRedoResponse struct {
	ID           string `json:"id"`
	CheckpointID string `json:"checkpoint_id"`
	State        string `json:"state"`
	DurationMs   int64  `json:"duration_ms"`
}

type DeleteCheckpointResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// =============================================================================
// Checkpoint Handlers
// =============================================================================

func handleCheckpointCreate(w http.ResponseWriter, r *http.Request, id string) {
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

	if sb.State != SandboxStateRunning && sb.State != SandboxStatePaused {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s is not in a checkpointable state", id))
		return
	}

	var req CreateCheckpointRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}
	}

	// Generate checkpoint ID
	cpID := fmt.Sprintf("cp_%s", uuid.New().String()[:8])
	if req.Name == "" {
		req.Name = fmt.Sprintf("checkpoint-%s", cpID[:8])
	}

	// Determine parent checkpoint (latest one)
	parentID := ""
	state.mu.RLock()
	if checkpoints, exists := state.checkpoints[id]; exists && len(checkpoints) > 0 {
		parentID = checkpoints[len(checkpoints)-1].ID
	}
	state.mu.RUnlock()

	// Create checkpoint directory
	cpDir := fmt.Sprintf("/var/lib/coco/checkpoints/%s/%s", id, cpID)
	if err := os.MkdirAll(cpDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create checkpoint directory: %v", err))
		return
	}

	start := time.Now()

	// TODO: Call cocovisor CHECKPOINT frame when real implementation ready
	// For now, simulate checkpoint creation

	// Create checkpoint metadata
	cp := Checkpoint{
		ID:        cpID,
		Name:      req.Name,
		SandboxID: id,
		CreatedAt: time.Now(),
		Path:      cpDir,
		SizeBytes: int64(sb.MemoryMB * 1024 * 1024), // Mock size
	}

	state.mu.Lock()
	if state.checkpoints[id] == nil {
		state.checkpoints[id] = []Checkpoint{}
	}
	state.checkpoints[id] = append(state.checkpoints[id], cp)
	state.mu.Unlock()

	duration := time.Since(start)
	log.Printf("Created checkpoint %s for sandbox %s (parent: %s, size: %d bytes, duration: %v)",
		cpID, id, parentID, cp.SizeBytes, duration)

	// Audit log checkpoint creation
	auditLogger.LogSandboxAction(AuditActionCheckpoint, AuditResultSuccess, id, r, nil)

	writeJSON(w, http.StatusCreated, CreateCheckpointResponse{
		ID:          cpID,
		Name:        req.Name,
		SandboxID:   id,
		CreatedAt:   cp.CreatedAt,
		Path:        cpDir,
		SizeBytes:   cp.SizeBytes,
		DurationMs:  duration.Milliseconds(),
		ParentID:    parentID,
		Description: req.Description,
	})
}

func handleCheckpointList(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	state.mu.RLock()
	_, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	state.mu.RLock()
	checkpoints := state.checkpoints[id]
	if checkpoints == nil {
		checkpoints = []Checkpoint{}
	}
	state.mu.RUnlock()

	// Sort by CreatedAt descending (newest first)
	sorted := make([]Checkpoint, len(checkpoints))
	copy(sorted, checkpoints)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].CreatedAt.Before(sorted[j].CreatedAt) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	writeJSON(w, http.StatusOK, ListCheckpointsResponse{
		Items:      sorted,
		TotalCount: len(sorted),
	})
}

func handleUndo(w http.ResponseWriter, r *http.Request, id string) {
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

	state.mu.RLock()
	checkpoints := state.checkpoints[id]
	state.mu.RUnlock()

	if len(checkpoints) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s has no checkpoints to undo", id))
		return
	}

	// Get the previous checkpoint (undo to the one before current)
	// For simplicity, undo to the first checkpoint in the chain
	targetCP := checkpoints[0]
	start := time.Now()

	// TODO: Call cocovisor RESTORE_CHECKPOINT frame when real implementation ready

	duration := time.Since(start)
	log.Printf("Undo sandbox %s to checkpoint %s (duration: %v)", id, targetCP.ID, duration)

	writeJSON(w, http.StatusOK, UndoRedoResponse{
		ID:           id,
		CheckpointID: targetCP.ID,
		State:        sb.State.String(),
		DurationMs:   duration.Milliseconds(),
	})
}

func handleRedo(w http.ResponseWriter, r *http.Request, id string) {
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

	state.mu.RLock()
	checkpoints := state.checkpoints[id]
	state.mu.RUnlock()

	if len(checkpoints) < 2 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s has no checkpoint chain to redo", id))
		return
	}

	// Redo to the next checkpoint in the chain (second one for now)
	targetCP := checkpoints[1]
	start := time.Now()

	// TODO: Call cocovisor RESTORE_CHECKPOINT frame when real implementation ready

	duration := time.Since(start)
	log.Printf("Redo sandbox %s to checkpoint %s (duration: %v)", id, targetCP.ID, duration)

	writeJSON(w, http.StatusOK, UndoRedoResponse{
		ID:           id,
		CheckpointID: targetCP.ID,
		State:        sb.State.String(),
		DurationMs:   duration.Milliseconds(),
	})
}

func handleCheckpointDelete(w http.ResponseWriter, r *http.Request, id string, checkpointID string) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	state.mu.RLock()
	checkpoints := state.checkpoints[id]
	state.mu.RUnlock()

	// Find and remove checkpoint
	found := false
	state.mu.Lock()
	defer state.mu.Unlock()

	newCheckpoints := make([]Checkpoint, 0, len(checkpoints))
	for _, cp := range checkpoints {
		if cp.ID == checkpointID {
			found = true
			// Clean up checkpoint directory
			os.RemoveAll(cp.Path)
			log.Printf("Deleted checkpoint %s for sandbox %s", checkpointID, id)
		} else {
			newCheckpoints = append(newCheckpoints, cp)
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Checkpoint %s not found", checkpointID))
		return
	}

	state.checkpoints[id] = newCheckpoints

	writeJSON(w, http.StatusOK, DeleteCheckpointResponse{
		Success: true,
		Message: fmt.Sprintf("Checkpoint %s deleted", checkpointID),
	})
}