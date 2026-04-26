// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/types"
)

// Manager handles checkpoint lifecycle
type Manager struct {
	basePath    string
	mu          sync.RWMutex
	checkpoints map[string]*types.Checkpoint
}

// NewManager creates a new checkpoint manager
func NewManager(basePath string) *Manager {
	return &Manager{
		basePath:    basePath,
		checkpoints: make(map[string]*types.Checkpoint),
	}
}

// Create creates a new checkpoint
func (m *Manager) Create(sandboxID, name, description string) (*types.Checkpoint, error) {
	id := fmt.Sprintf("%s-%d", sandboxID, time.Now().UnixNano())
	cp := &types.Checkpoint{
		ID:          id,
		SandboxID:   sandboxID,
		Name:        name,
		Description: description,
		Path:        filepath.Join(m.basePath, sandboxID, id),
		CreatedAt:   time.Now(),
		Compression: "zstd",
	}

	if err := os.MkdirAll(cp.Path, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoint dir: %w", err)
	}

	key := fmt.Sprintf("%s/%s", sandboxID, name)
	m.mu.Lock()
	m.checkpoints[key] = cp
	m.mu.Unlock()

	return cp, nil
}

// Get retrieves a checkpoint by sandbox ID and name
func (m *Manager) Get(sandboxID, name string) (*types.Checkpoint, error) {
	key := fmt.Sprintf("%s/%s", sandboxID, name)
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cp, ok := m.checkpoints[key]; ok {
		return cp, nil
	}
	return nil, fmt.Errorf("checkpoint not found")
}

// List returns all checkpoints for a sandbox
func (m *Manager) List(sandboxID string) []*types.Checkpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*types.Checkpoint
	for _, cp := range m.checkpoints {
		if cp.SandboxID == sandboxID {
			result = append(result, cp)
		}
	}
	return result
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