// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package coco

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	assert.Equal(t, "http://localhost:4747", client.baseURL)
	assert.Equal(t, 30*time.Second, client.timeout)
}

func TestNewClientWithOptions(t *testing.T) {
	client, err := NewClient(
		WithAddress("localhost:9999"),
		WithAPIKey("test-key"),
		WithTimeout(60*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, client)

	assert.Equal(t, "http://localhost:9999", client.baseURL)
	assert.Equal(t, "test-key", client.apiKey)
	assert.Equal(t, 60*time.Second, client.timeout)
}

func TestHealth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		json.NewEncoder(w).Encode(HealthResponse{
			Healthy: true,
			Version: "0.1.0",
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:])) // strip "http://"
	ctx := context.Background()

	resp, err := client.Health(ctx)
	require.NoError(t, err)
	assert.True(t, resp.Healthy)
	assert.Equal(t, "0.1.0", resp.Version)
}

func TestReady(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ready", r.URL.Path)

		json.NewEncoder(w).Encode(ReadyResponse{
			Ready: true,
			Checks: map[string]bool{
				"visor_socket": true,
				"badger_db":    true,
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.Ready(ctx)
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.True(t, resp.Checks["visor_socket"])
}

func TestCreateSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req SandboxConfig
		json.NewDecoder(r.Body).Decode(&req)

		json.NewEncoder(w).Encode(CreateSandboxResponse{
			ID:    "sb_test123",
			Name:  req.Name,
			State: SandboxStateCreating,
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.CreateSandbox(ctx, &SandboxConfig{
		Name:     "test-sandbox",
		Template: "alpine",
		MemoryMB: 512,
		VCPUs:    2,
	})
	require.NoError(t, err)
	assert.Equal(t, "sb_test123", resp.ID)
	assert.Equal(t, "test-sandbox", resp.Name)
}

func TestCreateSandboxWithDefaults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SandboxConfig
		json.NewDecoder(r.Body).Decode(&req)

		// Verify defaults were applied
		assert.Equal(t, "alpine", req.Template)
		assert.Equal(t, 512, req.MemoryMB)
		assert.Equal(t, 2, req.VCPUs)

		json.NewEncoder(w).Encode(CreateSandboxResponse{
			ID:    "sb_default",
			Name:  "default",
			State: SandboxStateCreating,
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	// Pass empty config - should apply defaults
	resp, err := client.CreateSandbox(ctx, &SandboxConfig{})
	require.NoError(t, err)
	assert.Equal(t, "sb_default", resp.ID)
}

func TestGetSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_abc123", r.URL.Path)

		json.NewEncoder(w).Encode(map[string]any{
			"sandbox": Sandbox{
				ID:       "sb_abc123",
				Name:     "test",
				State:    SandboxStateRunning,
				VsockCID: 3,
				PID:      10000,
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	sb, err := client.GetSandbox(ctx, "sb_abc123")
	require.NoError(t, err)
	assert.Equal(t, "sb_abc123", sb.ID)
	assert.Equal(t, SandboxStateRunning, sb.State)
	assert.Equal(t, uint32(3), sb.VsockCID)
}

func TestListSandboxes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes", r.URL.Path)

		json.NewEncoder(w).Encode(ListSandboxesResponse{
			Items: []Sandbox{
				{ID: "sb_1", Name: "sandbox-1", State: SandboxStateRunning},
				{ID: "sb_2", Name: "sandbox-2", State: SandboxStatePaused},
			},
			TotalCount: 2,
			Offset:     0,
			Limit:      100,
			HasMore:    false,
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.ListSandboxes(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(resp.Items))
	assert.Equal(t, 2, resp.TotalCount)
}

func TestListSandboxesWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		assert.Equal(t, "running", r.URL.Query().Get("state"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		json.NewEncoder(w).Encode(ListSandboxesResponse{
			Items:      []Sandbox{},
			TotalCount: 0,
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.ListSandboxes(ctx,
		WithStateFilter(SandboxStateRunning),
		WithOffset(0),
		WithLimit(10),
	)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}

func TestDestroySandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_destroy", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		json.NewEncoder(w).Encode(DestroyResponse{
			Success: true,
			Message: "Sandbox sb_destroy destroyed",
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.DestroySandbox(ctx, "sb_destroy")
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestForkSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_parent/fork", r.URL.Path)

		var req ForkRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "fork-name", req.Name)

		json.NewEncoder(w).Encode(ForkResponse{
			ID:           "sb_child",
			Name:         "fork-name",
			State:        SandboxStateRunning,
			ParentID:     "sb_parent",
			ForkDurationMs: 12,
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.ForkSandbox(ctx, "sb_parent", &ForkRequest{Name: "fork-name"})
	require.NoError(t, err)
	assert.Equal(t, "sb_child", resp.ID)
	assert.Equal(t, "sb_parent", resp.ParentID)
	assert.Equal(t, 12, resp.ForkDurationMs)
}

func TestHibernateSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_hibernate/hibernate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		json.NewEncoder(w).Encode(HibernateResponse{
			State:              SandboxStateHibernated,
			SnapshotID:         "snap_123",
			HibernationDurationMs: 1500,
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	resp, err := client.HibernateSandbox(ctx, "sb_hibernate")
	require.NoError(t, err)
	assert.Equal(t, SandboxStateHibernated, resp.State)
	assert.Equal(t, int64(1500), resp.HibernationDurationMs)
}

func TestResumeSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_resume/resume", r.URL.Path)

		json.NewEncoder(w).Encode(map[string]any{
			"id":    "sb_resume",
			"state": "running",
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	sb, err := client.ResumeSandbox(ctx, "sb_resume")
	require.NoError(t, err)
	assert.Equal(t, "sb_resume", sb.ID)
	assert.Equal(t, SandboxStateRunning, sb.State)
}

func TestPauseSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_pause/pause", r.URL.Path)

		json.NewEncoder(w).Encode(map[string]any{
			"id":    "sb_pause",
			"state": "paused",
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	sb, err := client.PauseSandbox(ctx, "sb_pause")
	require.NoError(t, err)
	assert.Equal(t, SandboxStatePaused, sb.State)
}

func TestCreateCheckpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_cp/checkpoints", r.URL.Path)

		var req struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "before-test", req.Name)
		assert.Equal(t, "Before running test", req.Description)

		json.NewEncoder(w).Encode(Checkpoint{
			ID:        "cp_001",
			Name:      "before-test",
			SandboxID: "sb_cp",
			CreatedAt: time.Now(),
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	cp, err := client.CreateCheckpoint(ctx, "sb_cp", "before-test", "Before running test")
	require.NoError(t, err)
	assert.Equal(t, "cp_001", cp.ID)
	assert.Equal(t, "before-test", cp.Name)
}

func TestListCheckpoints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_list/checkpoints", r.URL.Path)

		json.NewEncoder(w).Encode(map[string]any{
			"items": []Checkpoint{
				{ID: "cp_1", Name: "checkpoint-1"},
				{ID: "cp_2", Name: "checkpoint-2"},
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	cps, err := client.ListCheckpoints(ctx, "sb_list")
	require.NoError(t, err)
	assert.Equal(t, 2, len(cps))
}

func TestMetrics(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/metrics", r.URL.Path)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`# HELP coco_sandbox_count Number of sandboxes
# TYPE coco_sandbox_count gauge
coco_sandbox_count{state="running"} 5
`))
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	metrics, err := client.Metrics(ctx)
	require.NoError(t, err)
	assert.Contains(t, metrics, "coco_sandbox_count")
}

func TestAPIErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "SANDBOX_NOT_FOUND",
				"message": "Sandbox 'sb_notfound' not found",
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	_, err := client.GetSandbox(ctx, "sb_notfound")
	require.Error(t, err)

	apiErr, ok := err.(*Error)
	require.True(t, ok)
	assert.Equal(t, "SANDBOX_NOT_FOUND", apiErr.Code)
	assert.Contains(t, apiErr.Message, "sb_notfound")
}

func TestSandboxStateHelpers(t *testing.T) {
	assert.True(t, SandboxStateRunning.IsActive())
	assert.True(t, SandboxStateCreating.IsActive())
	assert.False(t, SandboxStatePaused.IsActive())
	assert.False(t, SandboxStateStopped.IsActive())
	assert.False(t, SandboxStateError.IsActive())

	assert.True(t, SandboxStateStopped.IsTerminal())
	assert.True(t, SandboxStateError.IsTerminal())
	assert.False(t, SandboxStateRunning.IsTerminal())
	assert.False(t, SandboxStateCreating.IsTerminal())
}

func TestUpdateSandbox(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sandboxes/sb_update", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var req struct {
			Name   string            `json:"name,omitempty"`
			Labels map[string]string `json:"labels,omitempty"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "updated-name", req.Name)
		assert.Equal(t, "value1", req.Labels["key1"])

		json.NewEncoder(w).Encode(map[string]any{
			"sandbox": Sandbox{
				ID:    "sb_update",
				Name:  "updated-name",
				State: SandboxStateRunning,
				Labels: map[string]string{"key1": "value1"},
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	ctx := context.Background()

	sb, err := client.UpdateSandbox(ctx, "sb_update", &SandboxConfig{
		Name:   "updated-name",
		Labels: map[string]string{"key1": "value1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "updated-name", sb.Name)
	assert.Equal(t, "value1", sb.Labels["key1"])
}

func TestWithLabelFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "tenant", r.URL.Query().Get("label_key"))
		assert.Equal(t, "acme", r.URL.Query().Get("label_value"))
		json.NewEncoder(w).Encode(ListSandboxesResponse{})
	}))
	defer ts.Close()

	client, _ := NewClient(WithAddress(ts.URL[7:]))
	_, err := client.ListSandboxes(context.Background(), WithLabelFilter("tenant", "acme"))
	require.NoError(t, err)
}
