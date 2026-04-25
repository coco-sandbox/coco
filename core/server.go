// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/coco-sandbox/coco/core/middleware"
)

// =============================================================================
// Router
// =============================================================================

func setupRoutes(mux *http.ServeMux) {
	// Health & Ready
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)
	mux.HandleFunc("/metrics", handleMetrics)

	// Sandboxes
	mux.HandleFunc("/v1/sandboxes", handleSandbox)
	mux.HandleFunc("/v1/sandboxes/", handleSandboxByID)

	// Cluster
	mux.HandleFunc("/cluster/nodes", handleClusterNodes)
	mux.HandleFunc("/cluster/nodes/", handleClusterNodeByID)
	mux.HandleFunc("/cluster/health", handleClusterHealth)

	// Raft
	mux.HandleFunc("/raft/status", handleRaftStatus)
	mux.HandleFunc("/raft/log", handleRaftLog)
	mux.HandleFunc("/raft/propose", handleRaftPropose)

	// Vsock Router (cross-node)
	mux.HandleFunc("/vsock/peers", handleVsockPeers)
	mux.HandleFunc("/vsock/peers/", handleVsockPeersByID)
	mux.HandleFunc("/vsock/connections", handleVsockConnections)
	mux.HandleFunc("/vsock/status", handleVsockStatus)

	// Note: /v1/sandboxes/{id}/exec, /fork, /hibernate, /resume,
	// /checkpoint, /checkpoints, /undo, /redo, /replay/start, /replay/stop
	// are handled by handleSandboxByID which strips the action suffix
}

// wrapMiddleware applies middleware chain to a handler
func wrapMiddleware(handler http.Handler) http.Handler {
	// 1. Request ID middleware (first - generates ID for tracing)
	handler = middleware.RequestID(handler)

	// 2. CORS middleware (handles preflight)
	handler = middleware.CORS(handler)

	// 3. Compression middleware (gzip)
	handler = middleware.Compression(handler)

	// 4. Timeout middleware (30 second default)
	handler = middleware.Timeout(30 * time.Second)(handler)

	// 5. Request logging (last - logs after request completes)
	logger := log.Default()
	handler = middleware.RequestLogging(logger)(handler)

	return handler
}

func handleSandbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleSandboxList(w, r)
	case http.MethodPost:
		handleSandboxCreate(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleSandboxByID(w http.ResponseWriter, r *http.Request) {
	id := sandboxIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Invalid sandbox ID")
		return
	}

	// Strip sandbox ID to get action path
	path := r.URL.Path
	prefix := "/v1/sandboxes/" + id

	// Handle subpaths: /v1/sandboxes/{id}/exec, /fork, /hibernate, etc.
	if len(path) > len(prefix) {
		action := path[len(prefix)+1:] // +1 for the slash

		switch action {
		case "exec":
			if r.Method == http.MethodPost {
				handleExec(w, r, id)
				return
			}
		case "fork":
			if r.Method == http.MethodPost {
				handleSandboxFork(w, r, id)
				return
			}
		case "pause":
			if r.Method == http.MethodPost {
				handleSandboxPause(w, r, id)
				return
			}
		case "hibernate":
			if r.Method == http.MethodPost {
				handleSandboxHibernate(w, r, id)
				return
			}
		case "resume":
			if r.Method == http.MethodPost {
				handleHibernateResume(w, r, id)
				return
			}
		case "checkpoint":
			if r.Method == http.MethodPost {
				handleCheckpointCreate(w, r, id)
				return
			}
		case "checkpoints":
			if r.Method == http.MethodGet {
				handleCheckpointList(w, r, id)
				return
			}
		case "undo":
			if r.Method == http.MethodPost {
				handleUndo(w, r, id)
				return
			}
		case "redo":
			if r.Method == http.MethodPost {
				handleRedo(w, r, id)
				return
			}
		case "replay/start":
			if r.Method == http.MethodPost {
				handleReplayStart(w, r, id)
				return
			}
		case "replay/stop":
			if r.Method == http.MethodPost {
				handleReplayStop(w, r, id)
				return
			}
		case "replay/events":
			if r.Method == http.MethodGet {
				handleReplayEvents(w, r, id)
				return
			}
		}

		// Check for checkpoints/{checkpoint_id} subpath
		if len(action) > 11 && action[:11] == "checkpoints/" {
			cpID := action[11:]
			if r.Method == http.MethodDelete {
				handleCheckpointDelete(w, r, id, cpID)
				return
			}
		}

		// Check for fs subpaths
		if len(action) > 3 && action[:3] == "fs/" {
			fsAction := action[3:]
			handleFS(w, r, id, fsAction)
			return
		}
	}

	// Default: delegate to appropriate handler based on method
	switch r.Method {
	case http.MethodGet:
		handleSandboxGet(w, r, id)
	case http.MethodDelete:
		handleSandboxDestroy(w, r, id)
	case http.MethodPatch:
		handleSandboxUpdate(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}