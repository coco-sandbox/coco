// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/coco-sandbox/coco/cmd/coco-gateway/middleware"
	"github.com/coco-sandbox/coco/pkg/api/handlers"
	"github.com/coco-sandbox/coco/pkg/types"
	"github.com/coco-sandbox/coco/pkg/visor"
)

func registerRoutes(mux *http.ServeMux, gw *GatewayServer, auth middleware.Authenticator, vp *visor.Pool) {
	// Sandbox handlers - gateway implements SandboxService interface
	sbHandler := handlers.NewSandboxHandler(gw)
	// Exec handler for streaming/interactive exec
	execHandler := handlers.NewExecHandler(vp, nil)

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
			handleExec(w, r, id, gw)
		case "streaming-exec":
			execHandler.HandleStreamingExec(w, r, id)
		case "interactive-exec":
			execHandler.HandleInteractiveExec(w, r, id)
		case "checkpoint":
			if r.Method == http.MethodPost {
				handleCreateCheckpoint(w, r, id, gw)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "restore":
			if r.Method == http.MethodPost {
				handleRestoreSandbox(w, r, id, gw)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "checkpoints":
			handleListCheckpoints(w, r, id, gw)
		default:
			http.NotFound(w, r)
		}
	})

	// Checkpoints (top-level - requires sandbox_id query param)
	mux.HandleFunc("/v1/checkpoints", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			sandboxID := r.URL.Query().Get("sandbox_id")
			if sandboxID == "" {
				http.Error(w, "sandbox_id query parameter required", http.StatusBadRequest)
				return
			}
			handleListCheckpoints(w, r, sandboxID, gw)
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
			handleGetCheckpoint(w, r, id, gw)
		case http.MethodDelete:
			handleDeleteCheckpoint(w, r, id, gw)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Templates - gateway implements TemplateService via adapter
	templateHandler := handlers.NewTemplateHandler(&templateServiceAdapter{gw: gw})
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

		action := parts[1]
		switch action {
		case "build":
			if r.Method == http.MethodPost {
				handleBuildTemplate(w, r, id, gw)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.NotFound(w, r)
		}
	})

	// Cluster
	mux.HandleFunc("/v1/cluster", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetClusterInfo(w, r, gw)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Nodes
	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleListNodes(w, r, gw)
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
				handleGetNode(w, r, id, gw)
			case http.MethodPost:
				handleDrainNode(w, r, id, gw)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if parts[1] == "drain" && r.Method == http.MethodPost {
			handleDrainNode(w, r, id, gw)
		} else {
			http.NotFound(w, r)
		}
	})
}

// Exec handler - proxies to master via GatewayServer
func handleExec(w http.ResponseWriter, r *http.Request, sandboxID string, gw *GatewayServer) {
	defer r.Body.Close()

	var req types.ExecSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}
	req.SandboxID = sandboxID

	output, exitCode, err := gw.ExecSandbox(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("exec failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stdout":   string(output),
		"exitCode": exitCode,
	})
}

// Cluster handlers
func handleGetClusterInfo(w http.ResponseWriter, r *http.Request, gw *GatewayServer) {
	info, err := gw.GetClusterInfo(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get cluster info: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleListNodes(w http.ResponseWriter, r *http.Request, gw *GatewayServer) {
	resp, err := gw.ListNodes(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list nodes: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": resp.Items,
		"total": resp.Total,
	})
}

func handleGetNode(w http.ResponseWriter, r *http.Request, id string, gw *GatewayServer) {
	node, err := gw.GetNode(r.Context(), id)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

func handleDrainNode(w http.ResponseWriter, r *http.Request, id string, gw *GatewayServer) {
	if err := gw.DrainNode(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to drain node: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "draining"})
}

// Checkpoint handlers
func handleCreateCheckpoint(w http.ResponseWriter, r *http.Request, sandboxID string, gw *GatewayServer) {
	defer r.Body.Close()

	var req types.CreateCheckpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	cp, err := gw.CreateCheckpoint(r.Context(), sandboxID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("create checkpoint failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cp)
}

func handleListCheckpoints(w http.ResponseWriter, r *http.Request, sandboxID string, gw *GatewayServer) {
	resp, err := gw.ListCheckpoints(r.Context(), sandboxID)
	if err != nil {
		http.Error(w, fmt.Sprintf("list checkpoints failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": resp.Items,
		"total": resp.Total,
	})
}

func handleGetCheckpoint(w http.ResponseWriter, r *http.Request, id string, gw *GatewayServer) {
	cp, err := gw.GetCheckpoint(r.Context(), id)
	if err != nil {
		http.Error(w, "checkpoint not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cp)
}

func handleDeleteCheckpoint(w http.ResponseWriter, r *http.Request, id string, gw *GatewayServer) {
	if err := gw.DeleteCheckpoint(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("delete checkpoint failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func handleRestoreSandbox(w http.ResponseWriter, r *http.Request, sandboxID string, gw *GatewayServer) {
	defer r.Body.Close()

	var req struct {
		CheckpointID string `json:"checkpoint_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if err := gw.RestoreSandbox(r.Context(), sandboxID, req.CheckpointID); err != nil {
		http.Error(w, fmt.Sprintf("restore failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restored"})
}

type templateServiceAdapter struct {
	gw *GatewayServer
}

func (a *templateServiceAdapter) List() ([]*handlers.Template, error) {
	ctx := context.Background()
	resp, err := a.gw.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	templates := make([]*handlers.Template, len(resp.Items))
	for i, t := range resp.Items {
		templates[i] = &handlers.Template{
			ID:        t.ID,
			Name:      t.Name,
			Rootfs:    t.RootfsPath,
			Kernel:    t.KernelPath,
			Initrd:    t.InitrdPath,
			MemoryMB:  uint32(t.MemoryMB),
			VCPUs:     uint32(t.VCPUs),
			CreatedAt: t.CreatedAt.Unix(),
		}
	}
	return templates, nil
}

func (a *templateServiceAdapter) Get(id string) (*handlers.Template, error) {
	ctx := context.Background()
	t, err := a.gw.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	return &handlers.Template{
		ID:        t.ID,
		Name:      t.Name,
		Rootfs:    t.RootfsPath,
		Kernel:    t.KernelPath,
		Initrd:    t.InitrdPath,
		MemoryMB:  uint32(t.MemoryMB),
		VCPUs:     uint32(t.VCPUs),
		CreatedAt: t.CreatedAt.Unix(),
	}, nil
}

func (a *templateServiceAdapter) Create(tpl *handlers.Template) (*handlers.Template, error) {
	ctx := context.Background()
	created, err := a.gw.CreateTemplate(ctx, &types.CreateTemplateRequest{
		Name:       tpl.Name,
		RootfsPath: tpl.Rootfs,
		KernelPath: tpl.Kernel,
		InitrdPath: tpl.Initrd,
	})
	if err != nil {
		return nil, err
	}
	return &handlers.Template{
		ID:        created.ID,
		Name:      created.Name,
		Rootfs:    created.RootfsPath,
		Kernel:    created.KernelPath,
		Initrd:    created.InitrdPath,
		MemoryMB:  uint32(created.MemoryMB),
		VCPUs:     uint32(created.VCPUs),
		CreatedAt: created.CreatedAt.Unix(),
	}, nil
}

func (a *templateServiceAdapter) Delete(id string) error {
	ctx := context.Background()
	return a.gw.DeleteTemplate(ctx, id)
}

func handleBuildTemplate(w http.ResponseWriter, r *http.Request, templateID string, gw *GatewayServer) {
	defer r.Body.Close()

	var req types.BuildTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if err := gw.BuildTemplate(r.Context(), templateID, &req); err != nil {
		http.Error(w, fmt.Sprintf("build template failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(types.BuildTemplateResponse{
		BuildID: "build_" + templateID,
		Status:  "building",
	})
}
