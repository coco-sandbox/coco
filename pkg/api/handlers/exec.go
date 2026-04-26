// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coco-sandbox/coco/pkg/visor"
)

// ExecHandler handles code execution requests
type ExecHandler struct {
	visorPool *visor.Pool
}

// NewExecHandler creates a new ExecHandler with the given visor connection pool
func NewExecHandler(visorPool *visor.Pool) *ExecHandler {
	return &ExecHandler{visorPool: visorPool}
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
	defer r.Body.Close()

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Convert handler request to visor request format
	envList := make([]string, 0, len(req.Env))
	for k, v := range req.Env {
		envList = append(envList, k+"="+v)
	}

	visorReq := visor.ExecRequest{
		Cmd:        req.Command,
		Args:       req.Args,
		Env:        envList,
		WorkingDir: req.WorkingDir,
	}

	// Acquire visor client from pool
	client, err := h.visorPool.Acquire()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to visor: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer h.visorPool.Release(client)

	// Collect output from visor
	var stdout, stderr []byte
	var exitCode uint32

	err = client.Exec(visorReq, func(chunk visor.ExecChunk) error {
		switch chunk.StreamType {
		case 1: // stdout
			stdout = append(stdout, chunk.Data...)
		case 2: // stderr
			stderr = append(stderr, chunk.Data...)
		case 3: // exit
			exitCode = chunk.ExitCode
		}
		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("exec failed: %v", err), http.StatusInternalServerError)
		return
	}

	resp := ExecResponse{
		Stdout:   string(stdout),
		Stderr:   string(stderr),
		ExitCode: int(exitCode),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleStreamingExec handles POST /v1/sandboxes/:id/streaming-exec
// This implements Server-Sent Events for streaming output
func (h *ExecHandler) HandleStreamingExec(w http.ResponseWriter, r *http.Request, sandboxID string) {
	defer r.Body.Close()

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

	// Convert handler request to visor request format
	envList := make([]string, 0, len(req.Env))
	for k, v := range req.Env {
		envList = append(envList, k+"="+v)
	}

	visorReq := visor.ExecRequest{
		Cmd:        req.Command,
		Args:       req.Args,
		Env:        envList,
		WorkingDir: req.WorkingDir,
	}

	// Acquire visor client from pool
	client, err := h.visorPool.Acquire()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to visor: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer h.visorPool.Release(client)

	// Stream output from visor using SSE format
	err = client.Exec(visorReq, func(chunk visor.ExecChunk) error {
		event := map[string]interface{}{
			"stream_type": chunk.StreamType,
			"data":        string(chunk.Data),
		}
		if chunk.StreamType == 3 {
			event["exit_code"] = chunk.ExitCode
		}

		data, err := json.Marshal(event)
		if err != nil {
			return nil // Skip malformed chunks
		}

		if chunk.StreamType == 3 {
			fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
		} else {
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		// Send error event
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
		flusher.Flush()
	}
}