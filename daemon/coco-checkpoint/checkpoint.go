// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package coco_checkpoint

import (
	"github.com/coco-sandbox/coco/pkg/checkpoint"
)

// CheckpointManager is an alias for the shared checkpoint manager implementation.
// See pkg/checkpoint/manager.go for the full implementation.
type CheckpointManager = checkpoint.CheckpointManager

// NewCheckpointManager creates a new checkpoint manager with the given checkpoint directory
// and optional metadata store. Pass nil for store to use a manager without persistent
// indexing (Create still works; List/Get/Delete require a store).
func NewCheckpointManager(dir string, store *checkpoint.Store) *CheckpointManager {
	return checkpoint.NewCheckpointManager(dir, store)
}
