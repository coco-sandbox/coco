// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coco-sandbox/coco/pkg/types"
)

// SandboxService defines the interface for sandbox operations
type SandboxService interface {
	Create(ctx context.Context, req *types.CreateSandboxRequest) (*types.Sandbox, error)
	Get(ctx context.Context, id string) (*types.Sandbox, error)
	List(ctx context.Context) ([]*types.Sandbox, error)
	Update(ctx context.Context, id string, sb *types.Sandbox) (*types.Sandbox, error)
	Delete(ctx context.Context, id string) error
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
	Fork(ctx context.Context, parentID string, req *types.ForkRequest) (*types.Sandbox, error)
	Hibernate(ctx context.Context, id string) error
	ResumeHibernate(ctx context.Context, id string) error
}

// SandboxHandler handles sandbox-related HTTP requests
type SandboxHandler struct {
	service SandboxService
}

// NewSandboxHandler creates a new SandboxHandler
func NewSandboxHandler(service SandboxService) *SandboxHandler {
	return &SandboxHandler{service: service}
}

// HandleCreate handles POST /v1/sandboxes
func (h *SandboxHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req types.CreateSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.MemoryMB < 0 {
		http.Error(w, "memory_mb must be non-negative", http.StatusBadRequest)
		return
	}
	if req.VCPUs < 0 {
		http.Error(w, "vcpus must be non-negative", http.StatusBadRequest)
		return
	}
	if req.Template == "" {
		req.Template = "alpine"
	}
	if req.MemoryMB == 0 {
		req.MemoryMB = 512
	}
	if req.VCPUs == 0 {
		req.VCPUs = 2
	}

	sb, err := h.service.Create(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sb)
}

// HandleList handles GET /v1/sandboxes
func (h *SandboxHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	sandboxes, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": sandboxes,
		"total": len(sandboxes),
	})
}

// HandleGet handles GET /v1/sandboxes/:id
func (h *SandboxHandler) HandleGet(w http.ResponseWriter, r *http.Request, id string) {
	sb, err := h.service.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "sandbox not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sb)
}

// HandleDelete handles DELETE /v1/sandboxes/:id
func (h *SandboxHandler) HandleDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.Delete(r.Context(), id); err != nil {
		if err.Error() == "sandbox not found" {
			http.Error(w, "sandbox not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("delete failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandlePause handles POST /v1/sandboxes/:id/pause
func (h *SandboxHandler) HandlePause(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.Pause(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"state": "paused"})
}

// HandleResume handles POST /v1/sandboxes/:id/resume
func (h *SandboxHandler) HandleResume(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.Resume(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"state": "running"})
}

// HandleFork handles POST /v1/sandboxes/:id/fork
func (h *SandboxHandler) HandleFork(w http.ResponseWriter, r *http.Request, id string) {
	var req types.ForkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	child, err := h.service.Fork(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(child)
}

// HandleHibernate handles POST /v1/sandboxes/:id/hibernate
func (h *SandboxHandler) HandleHibernate(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.Hibernate(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"state": "hibernated"})
}

// HandleResumeHibernate handles POST /v1/sandboxes/:id/resume-hibernate
func (h *SandboxHandler) HandleResumeHibernate(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.ResumeHibernate(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"state": "running"})
}