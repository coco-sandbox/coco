// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ExecHandler handles code execution requests
type ExecHandler struct{}

// NewExecHandler creates a new ExecHandler
func NewExecHandler() *ExecHandler {
	return &ExecHandler{}
}

// ExecRequest represents a request to execute code
type ExecRequest struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	WorkingDir string           `json:"working_dir"`
	TimeoutMs int64             `json:"timeout_ms"`
	Streaming bool              `json:"streaming"`
}

// ExecResponse represents the response from an exec operation
type ExecResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

// HandleExec handles POST /v1/sandboxes/:id/exec
func (h *ExecHandler) HandleExec(w http.ResponseWriter, r *http.Request, sandboxID string) {
	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	// In production, this would communicate with the visor daemon
	resp := ExecResponse{
		Stdout:     "mock output\n",
		Stderr:     "",
		ExitCode:   0,
		DurationMs: 100,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleStreamingExec handles POST /v1/sandboxes/:id/streaming-exec
// This implements Server-Sent Events for streaming output
func (h *ExecHandler) HandleStreamingExec(w http.ResponseWriter, r *http.Request, sandboxID string) {
	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send mock streaming output
	for i := 0; i < 5; i++ {
		data, _ := json.Marshal(map[string]interface{}{
			"stream_type": 1,
			"data":        fmt.Sprintf("chunk %d\n", i),
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Send exit event
	fmt.Fprintf(w, "event: done\ndata: {\"exit_code\": 0}\n\n")
	flusher.Flush()
}