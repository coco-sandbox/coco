// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/coco-sandbox/coco/pkg/api/handlers"
)

func registerRoutes(mux *http.ServeMux, gw *GatewayServer) {
	// Sandbox handlers - gateway implements SandboxService interface
	sbHandler := handlers.NewSandboxHandler(gw)

	// Exec handler - wired with checkpoint client
	execHandler := handlers.NewExecHandler(nil, nil)

	// Sandbox CRUD
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			sbHandler.HandleCreate(w, r)
		case http.MethodGet:
			sbHandler.HandleList(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/sandboxes/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/sandboxes/")
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				sbHandler.HandleGet(w, r, id)
			case http.MethodDelete:
				sbHandler.HandleDelete(w, r, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		action := parts[1]
		switch action {
		case "pause":
			sbHandler.HandlePause(w, r, id)
		case "resume":
			sbHandler.HandleResume(w, r, id)
		case "fork":
			sbHandler.HandleFork(w, r, id)
		case "hibernate":
			sbHandler.HandleHibernate(w, r, id)
		case "resume-hibernate":
			sbHandler.HandleResumeHibernate(w, r, id)
		case "exec":
			execHandler.HandleExec(w, r, id)
		case "streaming-exec":
			execHandler.HandleStreamingExec(w, r, id)
		case "interactive-exec":
			execHandler.HandleInteractiveExec(w, r, id)
		case "checkpoint":
			if r.Method == http.MethodPost {
				execHandler.HandleCreateCheckpoint(w, r, id)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "restore":
			if r.Method == http.MethodPost {
				execHandler.HandleRestoreSandbox(w, r, id)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "checkpoints":
			execHandler.HandleListCheckpoints(w, r, id)
		default:
			http.NotFound(w, r)
		}
	})

	// Checkpoints (top-level)
	mux.HandleFunc("/v1/checkpoints", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleListAllCheckpoints(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/checkpoints/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/checkpoints/")
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleGetCheckpoint(w, r, id)
		case http.MethodDelete:
			handleDeleteCheckpoint(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Templates - use pkg/api/handlers/template.go
	templateHandler := handlers.NewTemplateHandler(nil)
	mux.HandleFunc("/v1/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			templateHandler.HandleCreate(w, r)
		case http.MethodGet:
			templateHandler.HandleList(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/templates/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/templates/")
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				templateHandler.HandleGet(w, r, id)
			case http.MethodDelete:
				templateHandler.HandleDelete(w, r, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		http.NotFound(w, r)
	})

	// Cluster
	mux.HandleFunc("/v1/cluster", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetClusterInfo(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Nodes
	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleListNodes(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/nodes/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/nodes/")
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				handleGetNode(w, r, id)
			case http.MethodPost:
				handleDrainNode(w, r, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if parts[1] == "drain" && r.Method == http.MethodPost {
			handleDrainNode(w, r, id)
		} else {
			http.NotFound(w, r)
		}
	})
}

// Cluster handlers
func handleGetClusterInfo(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func handleListNodes(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func handleGetNode(w http.ResponseWriter, r *http.Request, id string) {
	_ = id
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func handleDrainNode(w http.ResponseWriter, r *http.Request, id string) {
	_ = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "draining"})
}

// Checkpoint handlers
func handleListAllCheckpoints(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func handleGetCheckpoint(w http.ResponseWriter, r *http.Request, id string) {
	_ = id
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func handleDeleteCheckpoint(w http.ResponseWriter, r *http.Request, id string) {
	_ = id
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
