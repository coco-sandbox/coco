// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package integration

import (
    "context"
    "testing"
    "time"

    "github.com/coco-sandbox/coco/pkg/api"
)

var testClient = api.NewClient("http://localhost:4747")

func TestSandboxCreateAndBoot(t *testing.T) {
    ctx := context.Background()

    // Create sandbox
    sb, err := testClient.Sandbox.Create(ctx, &api.CreateOptions{
        Template: "python-3.11",
        MemoryMB: 512,
        VCPUs:    2,
    })
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }
    defer sb.Delete(ctx)

    // Verify running state
    state, err := sb.State(ctx)
    if err != nil {
        t.Fatalf("State failed: %v", err)
    }
    if state != api.StateRunning {
        t.Errorf("Expected running, got %s", state)
    }
}

func TestSandboxExec(t *testing.T) {
    ctx := context.Background()
    sb, _ := testClient.Sandbox.Create(ctx, &api.CreateOptions{Template: "python-3.11"})
    defer sb.Delete(ctx)

    result, err := sb.Exec(ctx, &api.ExecRequest{
        Command: "echo hello",
    })
    if err != nil {
        t.Fatalf("Exec failed: %v", err)
    }
    if result.ExitCode != 0 {
        t.Errorf("Exit code != 0: %d", result.ExitCode)
    }
    if result.Stdout == "" {
        t.Errorf("Expected stdout, got empty")
    }
}

func TestSandboxPauseResume(t *testing.T) {
    ctx := context.Background()
    sb, _ := testClient.Sandbox.Create(ctx, &api.CreateOptions{Template: "python-3.11"})
    defer sb.Delete(ctx)

    err := sb.Pause(ctx)
    if err != nil {
        t.Fatalf("Pause failed: %v", err)
    }

    state, _ := sb.State(ctx)
    if state != api.StatePaused {
        t.Errorf("Expected paused, got %s", state)
    }

    err = sb.Resume(ctx)
    if err != nil {
        t.Fatalf("Resume failed: %v", err)
    }

    state, _ = sb.State(ctx)
    if state != api.StateRunning {
        t.Errorf("Expected running, got %s", state)
    }
}

func TestSandboxFork(t *testing.T) {
    ctx := context.Background()
    sb, _ := testClient.Sandbox.Create(ctx, &api.CreateOptions{Template: "python-3.11"})
    defer sb.Delete(ctx)

    forked, err := sb.Fork(ctx)
    if err != nil {
        t.Fatalf("Fork failed: %v", err)
    }
    defer forked.Delete(ctx)

    // Verify forked is running
    state, _ := forked.State(ctx)
    if state != api.StateRunning {
        t.Errorf("Fork not running: %s", state)
    }
}

func TestSandboxHibernate(t *testing.T) {
    ctx := context.Background()
    sb, _ := testClient.Sandbox.Create(ctx, &api.CreateOptions{Template: "python-3.11"})
    defer sb.Delete(ctx)

    err := sb.Hibernate(ctx)
    if err != nil {
        t.Fatalf("Hibernate failed: %v", err)
    }

    state, _ := sb.State(ctx)
    if state != api.StateHibernated {
        t.Errorf("Expected hibernated, got %s", state)
    }

    err = sb.Resume(ctx)
    if err != nil {
        t.Fatalf("Resume from hibernate failed: %v", err)
    }
}