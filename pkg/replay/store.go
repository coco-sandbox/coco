// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/coco-sandbox/coco/pkg/types"
)

// Store manages replay metadata
type Store struct {
	basePath string
	mu       sync.RWMutex
	index    map[string]*types.Checkpoint
}

// NewStore creates a new replay store
func NewStore(basePath string) (*Store, error) {
	s := &Store{basePath: basePath, index: make(map[string]*types.Checkpoint)}
	if err := s.loadIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

// Put saves a replay
func (s *Store) Put(r *types.Checkpoint) error {
	s.mu.Lock()
	s.index[r.ID] = r
	s.mu.Unlock()

	metaPath := filepath.Join(s.basePath, r.SandboxID, r.ID, "meta.json")
	os.MkdirAll(filepath.Dir(metaPath), 0755)
	data, _ := json.Marshal(r)
	return os.WriteFile(metaPath, data, 0644)
}

// Get retrieves a replay by ID
func (s *Store) Get(id string) (*types.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.index[id]; ok {
		return r, nil
	}
	return nil, ErrReplayNotFound
}

// ListBySandbox returns all replays for a sandbox
func (s *Store) ListBySandbox(sandboxID string) []*types.Checkpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*types.Checkpoint, 0)
	for _, r := range s.index {
		if r.SandboxID == sandboxID {
			out = append(out, r)
		}
	}
	return out
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
				var r types.Checkpoint
				if json.Unmarshal(data, &r) == nil {
					s.index[r.ID] = &r
				}
			}
		}
	}
	return nil
}

var ErrReplayNotFound = fmt.Errorf("replay not found")