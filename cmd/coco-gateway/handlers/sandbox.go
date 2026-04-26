package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coco-sandbox/coco/pkg/types"
)

type SandboxHandler struct {
	masterClient types.MasterClient
}

func NewSandboxHandler(masterClient types.MasterClient) *SandboxHandler {
	return &SandboxHandler{
		masterClient: masterClient,
	}
}

func (h *SandboxHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req types.CreateSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	timeoutCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := h.masterClient.CreateSandbox(timeoutCtx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	resp, err := h.masterClient.GetSandbox(r.Context(), &types.GetSandboxRequest{
		ID: id,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) List(w http.ResponseWriter, r *http.Request) {
	resp, err := h.masterClient.ListSandboxes(r.Context(), &types.ListSandboxesRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if err := h.masterClient.DeleteSandbox(r.Context(), &types.DeleteSandboxRequest{
		ID: id,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SandboxHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	resp, err := h.masterClient.StartSandbox(r.Context(), &types.StartSandboxRequest{
		ID: id,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if err := h.masterClient.StopSandbox(r.Context(), &types.StopSandboxRequest{
		ID: id,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SandboxHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	resp, err := h.masterClient.GetSandboxStats(r.Context(), &types.GetSandboxStatsRequest{
		ID: id,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
