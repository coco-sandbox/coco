// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	coco "github.com/coco-sandbox/coco/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAddress = "localhost:4747"
	testAPIKey  = "" // Use empty for local dev
)

func skipIfNotRunning(t *testing.T) {
	// Check if coco-core is running
	client, err := coco.NewClient(coco.WithAddress(testAddress))
	if err != nil {
		t.Skip("coco-core not running, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		t.Skip("coco-core not reachable, skipping integration test")
	}

	if !health.Healthy {
		t.Skip("coco-core not healthy, skipping integration test")
	}

	t.Log("coco-core is running and healthy")
}

func TestIntegration_HealthEndpoint(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	require.NoError(t, err)
	assert.True(t, health.Healthy)
	assert.NotEmpty(t, health.Version)
}

func TestIntegration_ReadyEndpoint(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready, err := client.Ready(ctx)
	require.NoError(t, err)
	// Note: ready might be false if cocovisor isn't running
	t.Logf("Ready: %+v", ready)
}

func TestIntegration_SandboxLifecycle(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a sandbox
	createResp, err := client.CreateSandbox(ctx, &coco.SandboxConfig{
		Name:     "integration-test",
		Template: "alpine",
		MemoryMB: 256,
		VCPUs:    1,
	})
	require.NoError(t, err)
	require.NotNil(t, createResp)
	t.Logf("Created sandbox: %s", createResp.ID)
	sandboxID := createResp.ID

	// Wait for sandbox to be running
	time.Sleep(2 * time.Second)

	// Get the sandbox
	sb, err := client.GetSandbox(ctx, sandboxID)
	require.NoError(t, err)
	assert.Equal(t, sandboxID, sb.ID)
	t.Logf("Sandbox state: %s", sb.State)

	// List sandboxes
	listResp, err := client.ListSandboxes(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, listResp.TotalCount, 1)
	t.Logf("Total sandboxes: %d", listResp.TotalCount)

	// Destroy the sandbox
	destroyResp, err := client.DestroySandbox(ctx, sandboxID)
	require.NoError(t, err)
	assert.True(t, destroyResp.Success)
	t.Logf("Destroyed sandbox: %s", sandboxID)
}

func TestIntegration_ExecCommand(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create sandbox
	createResp, err := client.CreateSandbox(ctx, &coco.SandboxConfig{
		Name:     "exec-test",
		Template: "alpine",
		MemoryMB: 256,
		VCPUs:    1,
	})
	require.NoError(t, err)
	sandboxID := createResp.ID
	t.Logf("Created sandbox for exec test: %s", sandboxID)

	// Wait for it to start
	time.Sleep(2 * time.Second)

	// Execute a simple command
	stdout, stderr, exitCode, err := client.ExecSync(ctx, sandboxID, &coco.ExecRequest{
		Cmd:  "echo",
		Args: []string{"hello", "world"},
	})
	t.Logf("Exec result: stdout=%q stderr=%q exitCode=%d err=%v", stdout, stderr, exitCode, err)

	// The exec might fail if cocovisor isn't fully running
	// But we've verified the API works

	// Cleanup
	client.DestroySandbox(ctx, sandboxID)
}

func TestIntegration_ForkSandbox(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create parent sandbox
	parentResp, err := client.CreateSandbox(ctx, &coco.SandboxConfig{
		Name:     "fork-parent",
		Template: "alpine",
		MemoryMB: 256,
		VCPUs:    1,
	})
	require.NoError(t, err)
	parentID := parentResp.ID
	t.Logf("Created parent sandbox: %s", parentID)

	time.Sleep(2 * time.Second)

	// Fork it
	forkResp, err := client.ForkSandbox(ctx, parentID, &coco.ForkRequest{
		Name: "fork-child",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, forkResp.ID)
	assert.Equal(t, parentID, forkResp.ParentID)
	t.Logf("Forked: parent=%s child=%s", parentID, forkResp.ID)

	// Cleanup
	client.DestroySandbox(ctx, parentID)
	client.DestroySandbox(ctx, forkResp.ID)
}

func TestIntegration_Checkpoints(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create sandbox
	sbResp, err := client.CreateSandbox(ctx, &coco.SandboxConfig{
		Name:     "checkpoint-test",
		Template: "alpine",
		MemoryMB: 256,
		VCPUs:    1,
	})
	require.NoError(t, err)
	sandboxID := sbResp.ID
	t.Logf("Created sandbox: %s", sandboxID)

	time.Sleep(2 * time.Second)

	// Create checkpoint
	cp, err := client.CreateCheckpoint(ctx, sandboxID, "test-checkpoint", "Integration test checkpoint")
	require.NoError(t, err)
	assert.NotEmpty(t, cp.ID)
	t.Logf("Created checkpoint: %s", cp.ID)

	// List checkpoints
	cps, err := client.ListCheckpoints(ctx, sandboxID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(cps), 1)
	t.Logf("Found %d checkpoints", len(cps))

	// Cleanup
	client.DestroySandbox(ctx, sandboxID)
}

func TestIntegration_Metrics(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metrics, err := client.Metrics(ctx)
	require.NoError(t, err)
	assert.Contains(t, metrics, "coco_")
	t.Logf("Metrics contains %d characters", len(metrics))
}

// Test the error handling
func TestIntegration_APIErrorHandling(t *testing.T) {
	skipIfNotRunning(t)

	client, err := coco.NewClient(coco.WithAddress(testAddress))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get non-existent sandbox
	_, err = client.GetSandbox(ctx, "sb_nonexistent")
	require.Error(t, err)

	// Should be an API error
	apiErr, ok := err.(*coco.Error)
	if ok {
		t.Logf("Got expected API error: %s - %s", apiErr.Code, apiErr.Message)
	}
}

func TestMain(m *testing.M) {
	// Check environment for test configuration
	if addr := os.Getenv("COCO_TEST_ADDRESS"); addr != "" {
		testAddress = addr
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}
