// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/coco-sandbox/coco/pkg/types"
)

// Store manages checkpoint metadata
type Store struct {
	basePath string
	mu       sync.RWMutex
	index    map[string]*types.Checkpoint
}

// NewStore creates a new checkpoint store
func NewStore(basePath string) (*Store, error) {
	s := &Store{basePath: basePath, index: make(map[string]*types.Checkpoint)}
	if err := s.loadIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

// Put saves a checkpoint
func (s *Store) Put(cp *types.Checkpoint) error {
	s.mu.Lock()
	s.index[cp.ID] = cp
	s.mu.Unlock()

	metaPath := filepath.Join(s.basePath, cp.SandboxID, cp.ID, "meta.json")
	os.MkdirAll(filepath.Dir(metaPath), 0755)
	data, _ := json.Marshal(cp)
	return os.WriteFile(metaPath, data, 0644)
}

// Get retrieves a checkpoint by ID
func (s *Store) Get(id string) (*types.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cp, ok := s.index[id]; ok {
		return cp, nil
	}
	return nil, ErrCheckpointNotFound
}

// List returns all checkpoints
func (s *Store) List() []*types.Checkpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*types.Checkpoint
	for _, cp := range s.index {
		result = append(result, cp)
	}
	return result
}

// Delete removes a checkpoint
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cp, ok := s.index[id]; ok {
		os.RemoveAll(cp.Path)
		delete(s.index, id)
	}
	return nil
}

func (s *Store) loadIndex() error {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			metaPath := filepath.Join(s.basePath, e.Name(), "meta.json")
			if data, err := os.ReadFile(metaPath); err == nil {
				var cp types.Checkpoint
				if json.Unmarshal(data, &cp) == nil {
					s.index[cp.ID] = &cp
				}
			}
		}
	}
	return nil
}

var ErrCheckpointNotFound = fmt.Errorf("checkpoint not found")