// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// =============================================================================
// Test Helper Functions
// =============================================================================

func TestSandboxStateString(t *testing.T) {
	tests := []struct {
		state    SandboxState
		expected string
	}{
		{SandboxStateCreating, "creating"},
		{SandboxStateRunning, "running"},
		{SandboxStatePaused, "paused"},
		{SandboxStateHibernated, "hibernated"},
		{SandboxStateStopping, "stopping"},
		{SandboxStateStopped, "stopped"},
		{SandboxStateError, "error"},
		{SandboxState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("SandboxState.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestSandboxIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/v1/sandboxes/sb_abc123", "sb_abc123"},
		{"/v1/sandboxes/sb_abc123/exec", "sb_abc123"},
		{"/v1/sandboxes/sb_abc123/fork", "sb_abc123"},
		{"/v1/sandboxes/sb_abc123/hibernate", "sb_abc123"},
		{"/v1/sandboxes/sb_abc123/checkpoints", "sb_abc123"},
		{"/v1/sandboxes/sb_abc123/checkpoints/cp_xyz789", "sb_abc123"},
		{"/v1/sandboxes/", ""},
		{"/v1/sandboxes", ""},
		{"/invalid", ""},
	}

	for _, tt := range tests {
		if got := sandboxIDFromPath(tt.path); got != tt.expected {
			t.Errorf("sandboxIDFromPath(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]any{"key": "value", "num": 42}

	writeJSON(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("writeJSON status = %v, want %v", rec.Code, http.StatusOK)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("writeJSON Content-Type = %v, want application/json", rec.Header().Get("Content-Type"))
	}

	var result map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Errorf("writeJSON decode error: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("writeJSON key = %v, want value", result["key"])
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusNotFound, "Sandbox not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("writeError status = %v, want %v", rec.Code, http.StatusNotFound)
	}

	var result map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Errorf("writeError decode error: %v", err)
	}

	errObj, ok := result["error"].(map[string]any)
	if !ok {
		t.Errorf("writeError error is not a map")
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasErr   bool
	}{
		{"0", 0, false},
		{"123", 123, false},
		{"999", 999, false},
		{"", 0, true},
		{"abc", 0, true},
		{"12a", 0, true},
		{"-5", 0, true},
	}

	for _, tt := range tests {
		got, err := parseInt(tt.input)
		if (err != nil) != tt.hasErr {
			t.Errorf("parseInt(%q) error = %v, want error = %v", tt.input, err, tt.hasErr)
		}
		if got != tt.expected {
			t.Errorf("parseInt(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// =============================================================================
// Error Response Tests
// =============================================================================

func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeSandboxNotFound,
		ErrCodeCheckpointNotFound,
		ErrCodeReplayNotFound,
		ErrCodeInvalidState,
		ErrCodeInvalidRequest,
		ErrCodeExecTimeout,
		ErrCodeForkDepthExceeded,
		ErrCodeHibernateFailed,
		ErrCodeResumeFailed,
		ErrCodeInternalError,
		ErrCodeRateLimited,
		ErrCodeUnauthorized,
	}

	for _, code := range codes {
		if code == "" {
			t.Errorf("ErrorCode is empty")
		}
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := newErrorResponse(ErrCodeSandboxNotFound, "Sandbox not found", "details")

	if resp.Error.Code != ErrCodeSandboxNotFound {
		t.Errorf("newErrorResponse.Code = %v, want %v", resp.Error.Code, ErrCodeSandboxNotFound)
	}
	if resp.Error.Message != "Sandbox not found" {
		t.Errorf("newErrorResponse.Message = %v, want %v", resp.Error.Message, "Sandbox not found")
	}
	if resp.Error.Details != "details" {
		t.Errorf("newErrorResponse.Details = %v, want %v", resp.Error.Details, "details")
	}
}

func TestWriteRateLimitedError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeRateLimitedError(rec, 30)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("writeRateLimitedError status = %v, want %v", rec.Code, http.StatusTooManyRequests)
	}

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter != "30" {
		t.Errorf("writeRateLimitedError Retry-After = %v, want 30", retryAfter)
	}
}

// =============================================================================
// Metrics Tests
// =============================================================================

func TestNewMetrics(t *testing.T) {
	m := newMetrics()
	if m == nil {
		t.Errorf("newMetrics() returned nil")
	}
	if m.sandboxesTotal == nil {
		t.Errorf("newMetrics() sandboxesTotal is nil")
	}
	if m.createsTotal == nil {
		t.Errorf("newMetrics() createsTotal is nil")
	}
}

// =============================================================================
// Rate Limiter Tests
// =============================================================================

func TestRateLimiterAllow(t *testing.T) {
	limiter := newRateLimiter(10, 5) // 10 rps, burst 5

	// First burst should be allowed
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.Allow("tenant1")
		if !allowed {
			t.Errorf("limiter.Allow(tenant1) attempt %d = false, want true", i+1)
		}
	}

	// Burst exhausted, should be denied
	allowed, retryAfter := limiter.Allow("tenant1")
	if allowed {
		t.Errorf("limiter.Allow(tenant1) after burst = true, want false")
	}
	if retryAfter < 1 {
		t.Errorf("limiter.Allow(tenant1) retryAfter = %v, want >= 1", retryAfter)
	}
}

// =============================================================================
// API Key Tests
// =============================================================================

func TestAPIKeyHasPermission(t *testing.T) {
	key := &APIKey{
		Key:      "test-key",
		Name:     "test",
		TenantID: "tenant1",
		Roles:    []Role{RoleOperator},
	}

	// Operator should have exec permission
	if !key.HasPermission("sandbox:exec") {
		t.Errorf("APIKey.HasPermission(sandbox:exec) = false, want true")
	}

	// Operator should NOT have admin permission
	if key.HasPermission("admin:write") {
		t.Errorf("APIKey.HasPermission(admin:write) = true, want false")
	}

	// Admin should have admin permission
	adminKey := &APIKey{
		Key:      "admin-key",
		Name:     "admin",
		TenantID: "tenant1",
		Roles:    []Role{RoleAdmin},
	}

	if !adminKey.HasPermission("admin:write") {
		t.Errorf("AdminKey.HasPermission(admin:write) = false, want true")
	}
}

// =============================================================================
// Context Tests
// =============================================================================

func TestContextHelpers(t *testing.T) {
	// Create API key
	apiKey := &APIKey{
		Key:      "test-key",
		TenantID: "tenant1",
		Roles:    []Role{RoleOperator},
	}

	// Test WithAPIKey and APIKeyFromContext
	ctx := WithAPIKey(nil, apiKey)
	if got := APIKeyFromContext(ctx); got != apiKey {
		t.Errorf("APIKeyFromContext() = %v, want %v", got, apiKey)
	}

	// Test WithTenantID and TenantIDFromContext
	ctx = WithTenantID(nil, "tenant123")
	if got := TenantIDFromContext(ctx); got != "tenant123" {
		t.Errorf("TenantIDFromContext() = %v, want %v", got, "tenant123")
	}

	// Test empty context
	if got := APIKeyFromContext(nil); got != nil {
		t.Errorf("APIKeyFromContext(nil) = %v, want nil", got)
	}
	if got := TenantIDFromContext(nil); got != "" {
		t.Errorf("TenantIDFromContext(nil) = %v, want empty string", got)
	}
}

// =============================================================================
// Checkpoint Tests
// =============================================================================

func TestCheckpointJSON(t *testing.T) {
	cp := Checkpoint{
		ID:        "cp_abc123",
		Name:      "test-checkpoint",
		SandboxID: "sb_xyz789",
		CreatedAt: time.Now(),
		Path:      "/var/lib/coco/checkpoints/sb_xyz789/cp_abc123",
		SizeBytes: 64 * 1024 * 1024,
	}

	data, err := json.Marshal(cp)
	if err != nil {
		t.Errorf("json.Marshal(Checkpoint) error: %v", err)
	}

	var decoded Checkpoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("json.Unmarshal(Checkpoint) error: %v", err)
	}

	if decoded.ID != cp.ID {
		t.Errorf("Checkpoint.ID = %v, want %v", decoded.ID, cp.ID)
	}
	if decoded.SizeBytes != cp.SizeBytes {
		t.Errorf("Checkpoint.SizeBytes = %v, want %v", decoded.SizeBytes, cp.SizeBytes)
	}
}

// =============================================================================
// Replay Tests
// =============================================================================

func TestReplayJSON(t *testing.T) {
	rp := Replay{
		ID:        "replay_abc123",
		SandboxID: "sb_xyz789",
		State:     "recording",
		StartTime: time.Now(),
	}

	data, err := json.Marshal(rp)
	if err != nil {
		t.Errorf("json.Marshal(Replay) error: %v", err)
	}

	var decoded Replay
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("json.Unmarshal(Replay) error: %v", err)
	}

	if decoded.ID != rp.ID {
		t.Errorf("Replay.ID = %v, want %v", decoded.ID, rp.ID)
	}
	if decoded.State != "recording" {
		t.Errorf("Replay.State = %v, want recording", decoded.State)
	}
}

// =============================================================================
// Hibernate Tests
// =============================================================================

func TestHibernateResponseJSON(t *testing.T) {
	resp := HibernateResponse{
		ID:            "sb_abc123",
		State:         "hibernated",
		HibernatePath: "/var/lib/coco/hibernation/sb_abc123",
		SizeBytes:      512 * 1024 * 1024,
		DurationMs:     1500,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("json.Marshal(HibernateResponse) error: %v", err)
	}

	var decoded HibernateResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("json.Unmarshal(HibernateResponse) error: %v", err)
	}

	if decoded.State != "hibernated" {
		t.Errorf("HibernateResponse.State = %v, want hibernated", decoded.State)
	}
	if decoded.DurationMs != 1500 {
		t.Errorf("HibernateResponse.DurationMs = %v, want 1500", decoded.DurationMs)
	}
}

// =============================================================================
// Resume Tests
// =============================================================================

func TestResumeResponseJSON(t *testing.T) {
	resp := ResumeResponse{
		ID:          "sb_abc123",
		State:       "running",
		DurationMs:  50,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("json.Marshal(ResumeResponse) error: %v", err)
	}

	var decoded ResumeResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("json.Unmarshal(ResumeResponse) error: %v", err)
	}

	if decoded.State != "running" {
		t.Errorf("ResumeResponse.State = %v, want running", decoded.State)
	}
	if decoded.DurationMs != 50 {
		t.Errorf("ResumeResponse.DurationMs = %v, want 50", decoded.DurationMs)
	}
}

// =============================================================================
// Test HTTP Handler Helpers
// =============================================================================

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleHealth status = %v, want %v", rec.Code, http.StatusOK)
	}
}

func TestReadyEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	handleReady(rec, req)

	// May return 503 if dependencies not available, but should be a valid response
	if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
		t.Errorf("handleReady status = %v, want OK or ServiceUnavailable", rec.Code)
	}
}

// =============================================================================
// Test HTTP Request Parsing
// =============================================================================

func TestParseJSONBody(t *testing.T) {
	body := `{"name": "test-sandbox", "template": "alpine", "memory_mb": 512, "vcpus": 2}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	var parsed struct {
		Name     string `json:"name"`
		Template string `json:"template"`
		MemoryMB int    `json:"memory_mb"`
		VCPUs    int    `json:"vcpus"`
	}

	if err := decodeJSON(req.Body, &parsed); err != nil {
		t.Errorf("decodeJSON() error: %v", err)
	}

	if parsed.Name != "test-sandbox" {
		t.Errorf("decodeJSON().Name = %v, want test-sandbox", parsed.Name)
	}
	if parsed.MemoryMB != 512 {
		t.Errorf("decodeJSON().MemoryMB = %v, want 512", parsed.MemoryMB)
	}
}

// =============================================================================
// Test Sandbox CRUD
// =============================================================================

func TestHandleSandboxCreateValidation(t *testing.T) {
	// Test with invalid JSON
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleSandboxCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("handleSandboxCreate with invalid JSON status = %v, want %v", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSandboxGetNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/sandboxes/sb_nonexistent", nil)
	rec := httptest.NewRecorder()

	handleSandboxGet(rec, req, "sb_nonexistent")

	if rec.Code != http.StatusNotFound {
		t.Errorf("handleSandboxGet not found status = %v, want %v", rec.Code, http.StatusNotFound)
	}
}

// =============================================================================
// Test exec.go constants
// =============================================================================

func TestStreamTypeConstants(t *testing.T) {
	if StreamTypeStdout != 1 {
		t.Errorf("StreamTypeStdout = %v, want 1", StreamTypeStdout)
	}
	if StreamTypeStderr != 2 {
		t.Errorf("StreamTypeStderr = %v, want 2", StreamTypeStderr)
	}
	if StreamTypeExit != 3 {
		t.Errorf("StreamTypeExit = %v, want 3", StreamTypeExit)
	}
	if StreamTypeSignal != 4 {
		t.Errorf("StreamTypeSignal = %v, want 4", StreamTypeSignal)
	}
}

func TestExecRequestJSON(t *testing.T) {
	req := ExecRequest{
		Cmd:        "ls",
		Args:       []string{"-la"},
		Env:        []string{"HOME=/root"},
		WorkingDir: "/tmp",
		TimeoutMs:  5000,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("json.Marshal(ExecRequest) error: %v", err)
	}

	var decoded ExecRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("json.Unmarshal(ExecRequest) error: %v", err)
	}

	if decoded.Cmd != "ls" {
		t.Errorf("ExecRequest.Cmd = %v, want ls", decoded.Cmd)
	}
	if len(decoded.Args) != 1 || decoded.Args[0] != "-la" {
		t.Errorf("ExecRequest.Args = %v, want [-la]", decoded.Args)
	}
}
