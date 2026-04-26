// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package integration

import (
    "context"
    "testing"

    "github.com/coco-sandbox/coco/pkg/api"
)

func TestSandboxCheckpoint(t *testing.T) {
    ctx := context.Background()
    sb, _ := testClient.Sandbox.Create(ctx, &api.CreateOptions{Template: "python-3.11"})
    defer sb.Delete(ctx)

    // Create checkpoint
    cp, err := sb.Checkpoint(ctx, "test-checkpoint", "before test")
    if err != nil {
        t.Fatalf("Checkpoint failed: %v", err)
    }
    if cp.Name != "test-checkpoint" {
        t.Errorf("Wrong checkpoint name: %s", cp.Name)
    }

    // List checkpoints
    cps, err := sb.ListCheckpoints(ctx)
    if err != nil {
        t.Fatalf("List checkpoints failed: %v", err)
    }
    if len(cps) != 1 {
        t.Errorf("Expected 1 checkpoint, got %d", len(cps))
    }
}