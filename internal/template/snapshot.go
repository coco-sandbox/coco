// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package template

import (
	"fmt"
	"os"
	"path/filepath"
)

type SnapshotState int

const (
	SnapshotReady SnapshotState = iota
	SnapshotBooting
	SnapshotCapturing
	SnapshotDone
	SnapshotFailed
)

type SnapshotManager struct {
	templateDir string
}

func NewSnapshotManager(templateDir string) *SnapshotManager {
	return &SnapshotManager{templateDir: templateDir}
}

func (sm *SnapshotManager) SnapshotTemplate(templateID string, vmPID int) error {
	snapDir := filepath.Join(sm.templateDir, templateID)
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	memPath := filepath.Join(snapDir, "snapshot.mem")
	statePath := filepath.Join(snapDir, "vmstate.bin")

	f1, err := os.Create(memPath)
	if err != nil {
		return fmt.Errorf("failed to create memory snapshot file: %w", err)
	}
	defer f1.Close()

	f2, err := os.Create(statePath)
	if err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}
	defer f2.Close()

	return nil
}

func (sm *SnapshotManager) RestoreSnapshot(templateID string) error {
	snapDir := filepath.Join(sm.templateDir, templateID)
	memPath := filepath.Join(snapDir, "snapshot.mem")
	statePath := filepath.Join(snapDir, "vmstate.bin")

	if _, err := os.Stat(memPath); err != nil {
		return fmt.Errorf("memory snapshot not found: %w", err)
	}
	if _, err := os.Stat(statePath); err != nil {
		return fmt.Errorf("VM state file not found: %w", err)
	}

	return nil
}