// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package integration

import (
    "context"
    "testing"

    "github.com/coco-sandbox/coco/pkg/api"
)

func TestReplayRecordAndPlayback(t *testing.T) {
    ctx := context.Background()
    sb, _ := testClient.Sandbox.Create(ctx, &api.CreateOptions{Template: "python-3.11"})
    defer sb.Delete(ctx)

    // Start recording
    err := sb.StartReplay(ctx, "test-replay")
    if err != nil {
        t.Fatalf("StartReplay failed: %v", err)
    }

    // Execute some commands
    sb.Exec(ctx, &api.ExecRequest{Command: "echo 1"})
    sb.Exec(ctx, &api.ExecRequest{Command: "echo 2"})

    // Stop recording
    replay, err := sb.StopReplay(ctx)
    if err != nil {
        t.Fatalf("StopReplay failed: %v", err)
    }
    if replay.Events < 2 {
        t.Errorf("Expected at least 2 events, got %d", replay.Events)
    }

    // Replay
    events, err := sb.ReplayEvents(ctx, replay.ID)
    if err != nil {
        t.Fatalf("ReplayEvents failed: %v", err)
    }
    if len(events) < 2 {
        t.Errorf("Expected at least 2 replay events, got %d", len(events))
    }
}