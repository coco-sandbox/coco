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
// Replay
// =============================================================================

func handleReplayStart(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	_, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	rp := Replay{
		ID:        fmt.Sprintf("replay_%s", uuid.New().String()[:8]),
		SandboxID: id,
		State:     "recording",
		StartTime: time.Now(),
		Path:      fmt.Sprintf("/var/lib/coco/replays/%s/events.log", id),
	}

	state.mu.Lock()
	state.replays[id] = rp
	state.mu.Unlock()

	log.Printf("Started replay recording %s for sandbox %s", rp.ID, id)
	writeJSON(w, http.StatusCreated, map[string]any{"replay": rp})
}

func handleReplayStop(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.Lock()
	rp, ok := state.replays[id]
	if ok {
		rp.State = "stopped"
		rp.StopTime = time.Now()
		state.replays[id] = rp
	}
	state.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Replay for sandbox %s not found", id))
		return
	}

	log.Printf("Stopped replay %s", rp.ID)
	writeJSON(w, http.StatusOK, map[string]any{"replay": rp})
}