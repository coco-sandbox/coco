package handlers

import (
	"context"
	"encoding/json"
	"net/http"
)

type SandboxHandler struct{}

func NewSandboxHandler() *SandboxHandler {
	return &SandboxHandler{}
}

func (h *SandboxHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = req
	_ = r

	resp := map[string]string{
		"id":     req.ID,
		"status": "created",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	_ = id

	resp := map[string]interface{}{
		"id":     id,
		"status": "running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) List(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"items": []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	_ = id

	w.WriteHeader(http.StatusNoContent)
}

func (h *SandboxHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	_ = id

	resp := map[string]string{
		"id":     id,
		"status": "running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	_ = id

	w.WriteHeader(http.StatusNoContent)
}

func (h *SandboxHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	_ = id

	resp := map[string]interface{}{
		"cpu":    0.0,
		"memory": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *SandboxHandler) HandleCreate(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *SandboxHandler) HandleGet(ctx context.Context, id string) (interface{}, error) {
	return nil, nil
}

func (h *SandboxHandler) HandleList(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (h *SandboxHandler) HandleDelete(ctx context.Context, id string) error {
	return nil
}
