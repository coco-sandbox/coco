package handlers

import (
	"context"
	"net/http"
)

// ExecHandler handles exec-related requests
type ExecHandler struct{}

// NewExecHandler creates a new ExecHandler
func NewExecHandler() *ExecHandler {
	return &ExecHandler{}
}

func (h *ExecHandler) Exec(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (h *ExecHandler) GetExecSession(ctx context.Context, sessionID string) (interface{}, error) {
	return nil, nil
}

func (h *ExecHandler) ResizeExec(ctx context.Context, sessionID string, width, height uint32) error {
	return nil
}

func (h *ExecHandler) SendExecInput(ctx context.Context, sessionID string, data []byte) error {
	return nil
}

func (h *ExecHandler) StreamExecOutput(ctx context.Context, sessionID string) (<-chan []byte, error) {
	ch := make(chan []byte)
	return ch, nil
}

func (h *ExecHandler) HandleExec(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *ExecHandler) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *ExecHandler) HandleResize(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *ExecHandler) HandleInput(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
