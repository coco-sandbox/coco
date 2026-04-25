// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coco-sandbox/coco/internal/types"
)

func TestCreateSandboxHandler(t *testing.T) {
	// Create a mock service for testing
	mockService := &mockSandboxService{
		sandboxes: make(map[string]*types.Sandbox),
	}
	handler := NewSandboxHandler(mockService)

	reqBody := `{
		"name": "test-sandbox",
		"template": "python-3.11",
		"memory_mb": 512,
		"vcpus": 2
	}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var resp types.Sandbox
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Name != "test-sandbox" {
		t.Errorf("Expected name 'test-sandbox', got '%s'", resp.Name)
	}
	if resp.Template != "python-3.11" {
		t.Errorf("Expected template 'python-3.11', got '%s'", resp.Template)
	}
}

func TestListSandboxesHandler(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: map[string]*types.Sandbox{
			"sb_1": {ID: "sb_1", Name: "sandbox-1", State: types.SandboxStateRunning},
			"sb_2": {ID: "sb_2", Name: "sandbox-2", State: types.SandboxStatePaused},
		},
	}
	handler := NewSandboxHandler(mockService)

	req := httptest.NewRequest("GET", "/v1/sandboxes", nil)
	w := httptest.NewRecorder()

	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []*types.Sandbox `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("Expected 2 sandboxes, got %d", resp.Total)
	}
}

func TestGetSandboxHandler(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: map[string]*types.Sandbox{
			"sb_test": {ID: "sb_test", Name: "test-sandbox", State: types.SandboxStateRunning},
		},
	}
	handler := NewSandboxHandler(mockService)

	req := httptest.NewRequest("GET", "/v1/sandboxes/sb_test", nil)
	w := httptest.NewRecorder()

	handler.HandleGet(w, req, "sb_test")

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestGetSandboxHandlerNotFound(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: make(map[string]*types.Sandbox),
	}
	handler := NewSandboxHandler(mockService)

	req := httptest.NewRequest("GET", "/v1/sandboxes/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.HandleGet(w, req, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestDeleteSandboxHandler(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: map[string]*types.Sandbox{
			"sb_test": {ID: "sb_test", Name: "test-sandbox"},
		},
	}
	handler := NewSandboxHandler(mockService)

	req := httptest.NewRequest("DELETE", "/v1/sandboxes/sb_test", nil)
	w := httptest.NewRecorder()

	handler.HandleDelete(w, req, "sb_test")

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}
}

func TestPauseSandboxHandler(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: map[string]*types.Sandbox{
			"sb_test": {ID: "sb_test", Name: "test-sandbox", State: types.SandboxStateRunning},
		},
	}
	handler := NewSandboxHandler(mockService)

	req := httptest.NewRequest("POST", "/v1/sandboxes/sb_test/pause", nil)
	w := httptest.NewRecorder()

	handler.HandlePause(w, req, "sb_test")

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestResumeSandboxHandler(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: map[string]*types.Sandbox{
			"sb_test": {ID: "sb_test", Name: "test-sandbox", State: types.SandboxStatePaused},
		},
	}
	handler := NewSandboxHandler(mockService)

	req := httptest.NewRequest("POST", "/v1/sandboxes/sb_test/resume", nil)
	w := httptest.NewRecorder()

	handler.HandleResume(w, req, "sb_test")

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestCreateSandboxHandlerBadRequest(t *testing.T) {
	mockService := &mockSandboxService{
		sandboxes: make(map[string]*types.Sandbox),
	}
	handler := NewSandboxHandler(mockService)

	// Invalid JSON
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// mockSandboxService implements SandboxService for testing
type mockSandboxService struct {
	sandboxes map[string]*types.Sandbox
}

func (m *mockSandboxService) Create(ctx context.Context, req *types.CreateSandboxRequest) (*types.Sandbox, error) {
	sb := &types.Sandbox{
		ID:       "sb_" + req.Name,
		Name:     req.Name,
		Template: req.Template,
		MemoryMB: req.MemoryMB,
		VCPUs:    req.VCPUs,
		Labels:   req.Labels,
		State:    types.SandboxStateRunning,
	}
	m.sandboxes[sb.ID] = sb
	return sb, nil
}

func (m *mockSandboxService) Get(ctx context.Context, id string) (*types.Sandbox, error) {
	sb, ok := m.sandboxes[id]
	if !ok {
		return nil, errNotFound
	}
	return sb, nil
}

func (m *mockSandboxService) List(ctx context.Context) ([]*types.Sandbox, error) {
	items := make([]*types.Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		items = append(items, sb)
	}
	return items, nil
}

func (m *mockSandboxService) Update(ctx context.Context, id string, sb *types.Sandbox) (*types.Sandbox, error) {
	m.sandboxes[id] = sb
	return sb, nil
}

func (m *mockSandboxService) Delete(ctx context.Context, id string) error {
	delete(m.sandboxes, id)
	return nil
}

func (m *mockSandboxService) Pause(ctx context.Context, id string) error {
	if sb, ok := m.sandboxes[id]; ok {
		sb.State = types.SandboxStatePaused
	}
	return nil
}

func (m *mockSandboxService) Resume(ctx context.Context, id string) error {
	if sb, ok := m.sandboxes[id]; ok {
		sb.State = types.SandboxStateRunning
	}
	return nil
}

func (m *mockSandboxService) Fork(ctx context.Context, parentID string, req *types.ForkRequest) (*types.Sandbox, error) {
	return &types.Sandbox{
		ID:       "sb_forked",
		Name:     req.Name,
		ParentID: parentID,
		State:    types.SandboxStateRunning,
	}, nil
}

func (m *mockSandboxService) Hibernate(ctx context.Context, id string) error {
	return nil
}

func (m *mockSandboxService) ResumeHibernate(ctx context.Context, id string) error {
	return nil
}

var errNotFound = errNotFoundImpl

type errNotFoundImpl struct{}

func (e errNotFoundImpl) Error() string { return "not found" }