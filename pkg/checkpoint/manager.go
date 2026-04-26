// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles checkpoint lifecycle
type Manager struct {
	basePath    string
	mu          sync.RWMutex
	checkpoints map[string]*Checkpoint // key: sandboxID/name
}

// NewManager creates a new checkpoint manager
func NewManager(basePath string) *Manager {
	return &Manager{
		basePath:    basePath,
		checkpoints: make(map[string]*Checkpoint),
	}
}

// Create creates a new checkpoint
func (m *Manager) Create(sandboxID, name, description string) (*Checkpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("cp_%s_%d", name, time.Now().UnixNano())
	cp := &Checkpoint{
		ID:          id,
		SandboxID:   sandboxID,
		Name:        name,
		Description: description,
		Path:        filepath.Join(m.basePath, sandboxID, id),
		CreatedAt:   time.Now(),
	}

	if err := os.MkdirAll(cp.Path, 0755); err != nil {
		return nil, err
	}

	// Create memory and state files
	f1, err := os.Create(filepath.Join(cp.Path, "memory.img"))
	if err != nil {
		return nil, err
	}
	f1.Close()

	f2, err := os.Create(filepath.Join(cp.Path, "vmstate.bin"))
	if err != nil {
		return nil, err
	}
	f2.Close()

	// In production: use clh-remote save-migration
	// clh-remote save-migration --vm-url unix:///run/coco/vm/{id}/sock --snapshot-path {cp.Path}

	if info, err := os.Stat(cp.Path); err == nil {
		cp.SizeBytes = info.Size()
	}

	key := fmt.Sprintf("%s/%s", sandboxID, name)
	m.checkpoints[key] = cp
	return cp, nil
}

// Restore restores a checkpoint
func (m *Manager) Restore(sandboxID, name string) error {
	key := fmt.Sprintf("%s/%s", sandboxID, name)
	m.mu.RLock()
	cp, ok := m.checkpoints[key]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("checkpoint not found: %s", name)
	}

	// In production: clh-remote restore-migration --snapshot-path {cp.Path}
	return nil
}

// List returns all checkpoints for a sandbox
func (m *Manager) List(sandboxID string) []*Checkpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Checkpoint, 0)
	for _, cp := range m.checkpoints {
		if cp.SandboxID == sandboxID {
			out = append(out, cp)
		}
	}
	return out
}

// Delete removes a checkpoint
func (m *Manager) Delete(sandboxID, name string) error {
	key := fmt.Sprintf("%s/%s", sandboxID, name)
	m.mu.Lock()
	defer m.mu.Unlock()

	if cp, ok := m.checkpoints[key]; ok {
		os.RemoveAll(cp.Path)
		delete(m.checkpoints, key)
	}
	return nil
}