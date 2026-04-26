// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package coco_checkpoint

import (
	"github.com/coco-sandbox/coco/pkg/checkpoint"
)

// CheckpointManager is an alias for the shared checkpoint manager implementation.
// See pkg/checkpoint/manager.go for the full implementation.
type CheckpointManager = checkpoint.CheckpointManager

// NewCheckpointManager creates a new checkpoint manager with the given checkpoint directory.
// Convenience constructor that delegates to pkg/checkpoint.NewCheckpointManager.
func NewCheckpointManager(dir string) *CheckpointManager {
	return checkpoint.NewCheckpointManager(dir)
}
