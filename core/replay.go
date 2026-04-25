// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Replay Request/Response Types
// =============================================================================

type StartReplayRequest struct {
	Name string `json:"name,omitempty"`
}

type StartReplayResponse struct {
	ID        string    `json:"id"`
	SandboxID string    `json:"sandbox_id"`
	State     string    `json:"state"`
	StartTime time.Time `json:"start_time"`
}

type StopReplayResponse struct {
	ID        string    `json:"id"`
	SandboxID string    `json:"sandbox_id"`
	State     string    `json:"state"`
	StopTime  time.Time `json:"stop_time"`
	Events    int       `json:"events"`
}

type ReplayEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"`
}

type ListReplayEventsResponse struct {
	ReplayID   string        `json:"replay_id"`
	SandboxID  string       `json:"sandbox_id"`
	Events     []ReplayEvent `json:"events"`
	TotalCount int          `json:"total_count"`
}

// =============================================================================
// Replay Handlers
// =============================================================================

func handleReplayStart(w http.ResponseWriter, r *http.Request, id string) {
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

	var req StartReplayRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("replay-%s", uuid.New().String()[:8])
	}

	replayID := fmt.Sprintf("replay_%s", uuid.New().String()[:8])
	startTime := time.Now()

	replay := Replay{
		ID:        replayID,
		SandboxID: id,
		State:     "recording",
		StartTime: startTime,
	}

	state.mu.Lock()
	state.replays[replayID] = replay
	state.mu.Unlock()

	log.Printf("Started replay %s for sandbox %s", replayID, id)

	writeJSON(w, http.StatusCreated, StartReplayResponse{
		ID:        replayID,
		SandboxID: id,
		State:     replay.State,
		StartTime: startTime,
	})
}

func handleReplayStop(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
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

	// Find active replay for this sandbox
	state.mu.RLock()
	var activeReplay Replay
	found := false
	for _, rep := range state.replays {
		if rep.SandboxID == id && rep.State == "recording" {
			activeReplay = rep
			found = true
			break
		}
	}
	state.mu.RUnlock()

	if !found {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("No active replay for sandbox %s", id))
		return
	}

	stopTime := time.Now()
	events := 0 // Mock event count

	state.mu.Lock()
	if rep, ok := state.replays[activeReplay.ID]; ok {
		rep.State = "stopped"
		rep.StopTime = stopTime
		rep.Events = events
		state.replays[activeReplay.ID] = rep
	}
	state.mu.Unlock()

	log.Printf("Stopped replay %s for sandbox %s (events: %d)", activeReplay.ID, id, events)

	writeJSON(w, http.StatusOK, StopReplayResponse{
		ID:        activeReplay.ID,
		SandboxID: id,
		State:     "stopped",
		StopTime:  stopTime,
		Events:    events,
	})
}

func handleReplayEvents(w http.ResponseWriter, r *http.Request, id string) {
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

	// Find replay for this sandbox
	state.mu.RLock()
	var replay Replay
	found := false
	for _, rep := range state.replays {
		if rep.SandboxID == id {
			replay = rep
			found = true
			break
		}
	}
	state.mu.RUnlock()

	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("No replay found for sandbox %s", id))
		return
	}

	// Mock events - in real implementation, read from event store
	events := []ReplayEvent{
		{
			Type:      "exec",
			Timestamp: replay.StartTime,
			Data:      `{"command": "echo hello", "exit_code": 0}`,
		},
		{
			Type:      "fork",
			Timestamp: replay.StartTime.Add(1 * time.Second),
			Data:      `{"parent_id": "` + id + `", "child_id": "sb_child1"}`,
		},
	}

	writeJSON(w, http.StatusOK, ListReplayEventsResponse{
		ReplayID:   replay.ID,
		SandboxID:  id,
		Events:     events,
		TotalCount: len(events),
	})
}