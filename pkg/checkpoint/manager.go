// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package checkpoint

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/coco-sandbox/coco/pkg/types"
)

// CheckpointManager creates and manages VM checkpoints per spec §5.
type CheckpointManager struct {
	dir        string
	compressor Compressor
	store      *Store
}

// Compressor defines the interface for checkpoint compression.
type Compressor interface {
	CompressFile(dst, src string) error
	DecompressFile(dst, src string) error
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(dir string, store *Store) *CheckpointManager {
	return &CheckpointManager{
		dir:   dir,
		store: store,
	}
}

// SetCompressor sets the compression implementation.
func (cm *CheckpointManager) SetCompressor(c Compressor) {
	cm.compressor = c
}

// Create creates a new checkpoint for a sandbox.
// It creates the checkpoint directory structure and metadata per spec §5.3.
func (cm *CheckpointManager) Create(ctx context.Context, sandboxID, name string) (*types.Checkpoint, error) {
	id := fmt.Sprintf("cp-%d", len(sandboxID)+len(name))
	path := filepath.Join(cm.dir, sandboxID, id)

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoint dir: %w", err)
	}

	cp := &types.Checkpoint{
		ID:        id,
		SandboxID: sandboxID,
		Name:      name,
		Path:      path,
		CreatedAt: time.Now(),
	}

	// Save to store if available
	if cm.store != nil {
		if err := cm.store.Put(cp); err != nil {
			return nil, fmt.Errorf("save checkpoint: %w", err)
		}
	}

	return cp, nil
}

// List returns all checkpoints for a sandbox.
func (cm *CheckpointManager) List(sandboxID string) ([]*types.Checkpoint, error) {
	if cm.store == nil {
		return nil, fmt.Errorf("no store configured")
	}
	all := cm.store.List()
	var result []*types.Checkpoint
	for _, cp := range all {
		if cp.SandboxID == sandboxID {
			result = append(result, cp)
		}
	}
	return result, nil
}

// Get retrieves a checkpoint by ID.
func (cm *CheckpointManager) Get(id string) (*types.Checkpoint, error) {
	if cm.store == nil {
		return nil, fmt.Errorf("no store configured")
	}
	return cm.store.Get(id)
}

// Delete removes a checkpoint.
func (cm *CheckpointManager) Delete(id string) error {
	if cm.store == nil {
		return fmt.Errorf("no store configured")
	}
	cp, err := cm.store.Get(id)
	if err != nil {
		return err
	}
	// Remove checkpoint files
	if err := os.RemoveAll(cp.Path); err != nil {
		return fmt.Errorf("remove checkpoint files: %w", err)
	}
	return cm.store.Delete(id)
}

// HasCheckpoint returns true if a checkpoint exists for the sandbox.
func (cm *CheckpointManager) HasCheckpoint(sandboxID string) (bool, error) {
	checkpoints, err := cm.List(sandboxID)
	if err != nil {
		return false, err
	}
	return len(checkpoints) > 0, nil
}

// Restore restores a sandbox from its latest checkpoint on the specified node.
// This is a stub - actual restoration requires CRIU integration.
func (cm *CheckpointManager) Restore(ctx context.Context, sandboxID, nodeID string) error {
	checkpoints, err := cm.List(sandboxID)
	if err != nil {
		return fmt.Errorf("list checkpoints: %w", err)
	}
	if len(checkpoints) == 0 {
		return fmt.Errorf("no checkpoints found for sandbox %s", sandboxID)
	}

	latest := checkpoints[len(checkpoints)-1]
	log.Printf("Restoring sandbox %s from checkpoint %s on node %s", sandboxID, latest.ID, nodeID)
	// Actual CRIU restore would go here - requires kernel CRIU support
	return nil
}

// MemoryImagePath returns the full path for a checkpoint's memory image.
func MemoryImagePath(cp *types.Checkpoint) string {
	return filepath.Join(cp.Path, "memory.img.zst")
}

// CPUStatePath returns the full path for the CPU state file.
func CPUStatePath(cp *types.Checkpoint) string {
	return filepath.Join(cp.Path, "cpu.dat")
}

// MetadataPath returns the full path for the metadata file.
func MetadataPath(cp *types.Checkpoint) string {
	return filepath.Join(cp.Path, "meta.json")
}
