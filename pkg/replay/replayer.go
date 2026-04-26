// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package replay

import (
	"encoding/json"
	"os"
	"path/filepath"

	"coco/pkg/types"
)

// Replayer replays recorded sessions with step control
type Replayer struct {
	sandboxID string
	events    []types.ReplayEvent
	idx       int
}

// NewReplayer creates a new replayer from recorded events
func NewReplayer(sandboxID, basePath string) (*Replayer, error) {
	eventsPath := filepath.Join(basePath, sandboxID, "replays", "events.json")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		return nil, err
	}

	var events []types.ReplayEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}

	return &Replayer{sandboxID: sandboxID, events: events, idx: 0}, nil
}

// Next returns the next event and whether more exist
func (r *Replayer) Next() (*types.ReplayEvent, bool) {
	if r.idx >= len(r.events) {
		return nil, false
	}
	ev := r.events[r.idx]
	r.idx++
	return &ev, true
}

// Reset restarts playback from the beginning
func (r *Replayer) Reset() {
	r.idx = 0
}

// EventCount returns total number of events
func (r *Replayer) EventCount() int {
	return len(r.events)
}
