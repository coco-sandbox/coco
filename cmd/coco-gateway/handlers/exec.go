package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/coco-sandbox/coco/pkg/types"
)

type ExecHandler struct {
	masterClient types.MasterClient
}

func NewExecHandler(masterClient types.MasterClient) *ExecHandler {
	return &ExecHandler{
		masterClient: masterClient,
	}
}

func (h *ExecHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req types.ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.masterClient.Exec(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ExecHandler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	resp, err := h.masterClient.GetExecSession(r.Context(), &types.GetExecSessionRequest{
		SessionID: sessionID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ExecHandler) Resize(w http.ResponseWriter, r *http.Request) {
	var req types.ResizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.masterClient.ResizeExec(r.Context(), &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ExecHandler) SendInput(w http.ResponseWriter, r *http.Request) {
	var req types.ExecInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.masterClient.SendExecInput(r.Context(), &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ExecHandler) StreamOutput(ctx context.Context, sessionID string, ch chan []byte) error {
	return h.masterClient.StreamExecOutput(ctx, sessionID, ch)
}
