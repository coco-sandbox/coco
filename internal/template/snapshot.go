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
	os.MkdirAll(snapDir, 0755)

	memPath := filepath.Join(snapDir, "snapshot.mem")
	statePath := filepath.Join(snapDir, "vmstate.bin")

	f1, _ := os.Create(memPath)
	defer f1.Close()
	f2, _ := os.Create(statePath)
	defer f2.Close()

	return nil
}

func (sm *SnapshotManager) RestoreSnapshot(templateID string) error {
	snapDir := filepath.Join(sm.templateDir, templateID)
	memPath := filepath.Join(snapDir, "snapshot.mem")

	if _, err := os.Stat(memPath); err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}

	return nil
}