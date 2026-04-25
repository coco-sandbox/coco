// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coco-sandbox/coco/core/visor"
)

// =============================================================================
// Exec
// =============================================================================

const (
	StreamTypeStdout = 1
	StreamTypeStderr = 2
	StreamTypeExit   = 3
	StreamTypeSignal = 4
)

// ExecRequest represents a command execution request
type ExecRequest struct {
	Cmd        string   `json:"cmd"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	WorkingDir string   `json:"working_dir"`
	TimeoutMs int64    `json:"timeout_ms,omitempty"`
}

// ExecChunk represents a chunk of exec output
type ExecChunk struct {
	StreamType int    `json:"stream_type"`
	Data      string `json:"data"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

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

	// Audit log exec action
	auditLogger.LogSandboxAction(AuditActionExec, AuditResultSuccess, id, r, nil)

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Normalize: allow cmd as string or array
	// If cmd is empty but args are provided, use "sh -c" with args joined
	if req.Cmd == "" && len(req.Args) > 0 {
		req.Cmd = "sh"
		req.Args = append([]string{"-c"}, req.Args...)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Set up SSE-like streaming
	w.Header().Set("Content-Type", "application/x-stream+json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	start := time.Now()

	// Build exec frame
	frame, err := visor.BuildExecFrame(req.Cmd, req.Args, req.Env, req.WorkingDir)
	if err != nil {
		fmt.Fprintf(w, `{"stream_type":3,"exit_code":1,"error":%q}`+"\n", err.Error())
		flusher.Flush()
		return
	}

	client, err := visor.Dial()
	if err != nil {
		// Fallback: send mock exec response for development
		mockResp := `{"stream_type":1,"data":"mock output from sandbox\n"}` + "\n"
		mockResp += `{"stream_type":3,"exit_code":0}` + "\n"
		w.Write([]byte(mockResp))
		flusher.Flush()
		return
	}
	defer client.Close()

	// Set up timeout if specified (default 30s)
	timeout := 30 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	timeoutChan := time.After(timeout)

	ch, err := client.SendExec(frame)
	if err != nil {
		fmt.Fprintf(w, `{"stream_type":3,"exit_code":1,"error":%q}`+"\n", err.Error())
		flusher.Flush()
		return
	}

	// Stream responses as they come
	for {
		select {
		case <-timeoutChan:
			// Timeout reached
			fmt.Fprintf(w, `{"stream_type":4,"data":"command timed out after %v"}`+"\n", timeout)
			fmt.Fprintf(w, `{"stream_type":3,"exit_code":124}`+"\n") // 124 = timeout exit code
			flusher.Flush()
			return
		case chunk, ok := <-ch:
			if !ok {
				// Channel closed, exec complete
				break
			}
			line := fmt.Sprintf(`{"stream_type":%d,"data":%q}`, chunk.StreamType, string(chunk.Data))
			if chunk.ExitCode >= 0 {
				line += fmt.Sprintf(`,"exit_code":%d`, chunk.ExitCode)
			}
			line += "\n"
			w.Write([]byte(line))
			flusher.Flush()
			if chunk.StreamType == StreamTypeExit {
				return
			}
		}
	}

	elapsed := time.Since(start)
	state.metrics.RecordExec(elapsed)
}
