// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Recorder captures exec events during sandbox lifecycle
type Recorder struct {
	sandboxID string
	events    []ReplayEvent
	mu        sync.Mutex
	path      string
	state     string
}

// NewRecorder creates a new recorder for a sandbox
func NewRecorder(sandboxID, basePath string) *Recorder {
	path := filepath.Join(basePath, sandboxID, "replays")
	os.MkdirAll(path, 0755)
	return &Recorder{
		sandboxID: sandboxID,
		events:    make([]ReplayEvent, 0),
		path:      path,
		state:     "recording",
	}
}

// RecordEvent records a single event
func (r *Recorder) RecordEvent(eventType string, data interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	r.events = append(r.events, ReplayEvent{
		Type:      eventType,
		Timestamp: time.Now().UnixNano(),
		Data:      string(jsonData),
	})
	return nil
}

// Stop stops recording and saves events
func (r *Recorder) Stop() error {
	r.mu.Lock()
	r.state = "stopped"
	r.mu.Unlock()

	return r.save()
}

func (r *Recorder) save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventsPath := filepath.Join(r.path, "events.json")
	data, err := json.Marshal(r.events)
	if err != nil {
		return err
	}

	return os.WriteFile(eventsPath, data, 0644)
}

// State returns the current recording state
func (r *Recorder) State() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}