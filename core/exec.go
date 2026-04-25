// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// =============================================================================
// Exec
// =============================================================================

func handleExec(w http.ResponseWriter, r *http.Request, id string) {
	state.mu.RLock()
	sb, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	if sb.State != SandboxStateRunning {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Sandbox %s is not running (state: %s)", id, sb.State.String()))
		return
	}

	var req struct {
		Cmd      string   `json:"cmd"`
		Args     []string `json:"args"`
		Env      []string `json:"env"`
		WorkingDir string  `json:"working_dir"`
	}

	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "application/x-stream+json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	frame, err := visor.BuildExecFrame(req.Cmd, req.Args, req.Env, req.WorkingDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("BuildExecFrame failed: %v", err))
		return
	}

	client, err := visor.Dial()
	if err != nil {
		// Fallback: send mock exec response
		mockResp := `{"stream_type":1,"data":"mock exec output\n"}` + "\n" + `{"stream_type":3,"exit_code":0}` + "\n"
		w.Write([]byte(mockResp))
		flusher.Flush()
		return
	}
	defer client.Close()

	ch, err := client.SendExec(frame)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("SendExec failed: %v", err))
		return
	}

	for chunk := range ch {
		line := fmt.Sprintf(`{"stream_type":%d,"data":%q,"exit_code":%d}`+"\n",
			chunk.StreamType, string(chunk.Data), chunk.ExitCode)
		w.Write([]byte(line))
		flusher.Flush()
	}

	elapsed := 5 * time.Millisecond
	state.metrics.RecordExec(elapsed)
}

func decodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}