// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coco-sandbox/coco/pkg/api"
)

// Template represents a VM template
type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Rootfs    string `json:"rootfs"`
	Kernel    string `json:"kernel"`
	Initrd    string `json:"initrd,omitempty"`
	MemoryMB  uint32 `json:"memory_mb"`
	VCPUs     uint32 `json:"vcpus"`
	CreatedAt int64  `json:"created_at"`
}

// TemplateService defines the interface for template operations
type TemplateService interface {
	List() ([]*Template, error)
	Get(id string) (*Template, error)
	Create(tpl *Template) (*Template, error)
	Delete(id string) error
}

// TemplateHandler handles template-related HTTP requests
type TemplateHandler struct {
	service TemplateService
}

// NewTemplateHandler creates a new TemplateHandler
func NewTemplateHandler(service TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

// HandleList handles GET /v1/templates
func (h *TemplateHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	templates, err := h.service.List()
	if err != nil {
		api.WriteInternalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": templates,
		"total": len(templates),
	})
}

// HandleGet handles GET /v1/templates/:id
func (h *TemplateHandler) HandleGet(w http.ResponseWriter, r *http.Request, id string) {
	tpl, err := h.service.Get(id)
	if err != nil {
		api.WriteNotFound(w, "template not found")
		return
	}

	writeJSON(w, http.StatusOK, tpl)
}

// HandleCreate handles POST /v1/templates
func (h *TemplateHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Rootfs   string `json:"rootfs"`
		Kernel   string `json:"kernel"`
		Initrd   string `json:"initrd,omitempty"`
		MemoryMB uint32 `json:"memory_mb"`
		VCPUs    uint32 `json:"vcpus"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Name == "" || req.Rootfs == "" || req.Kernel == "" {
		api.WriteBadRequest(w, "name, rootfs, and kernel are required")
		return
	}

	if req.MemoryMB == 0 {
		req.MemoryMB = 512
	}
	if req.VCPUs == 0 {
		req.VCPUs = 2
	}

	id := fmt.Sprintf("tpl_%s_%d", req.Name, time.Now().UnixNano())
	tpl := &Template{
		ID:        id,
		Name:      req.Name,
		Rootfs:    req.Rootfs,
		Kernel:    req.Kernel,
		Initrd:    req.Initrd,
		MemoryMB:  req.MemoryMB,
		VCPUs:     req.VCPUs,
		CreatedAt: time.Now().Unix(),
	}

	created, err := h.service.Create(tpl)
	if err != nil {
		api.WriteInternalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// HandleDelete handles DELETE /v1/templates/:id
func (h *TemplateHandler) HandleDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.Delete(id); err != nil {
		api.WriteInternalError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
