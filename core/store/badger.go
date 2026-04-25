// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// BadgerDB-compatible interface for sandbox state.
// For v1, we use a simple JSON file store.
// Production deployments can swap this for etcd/Consul.

type Store struct {
	mu      sync.RWMutex
	baseDir string
	data    map[string][]byte
}

func New(baseDir string) (*Store, error) {
	os.MkdirAll(baseDir, 0755)
	s := &Store{
		baseDir: baseDir,
		data:    make(map[string][]byte),
	}
	s.loadAll()
	return s, nil
}

func (s *Store) loadAll() {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(s.baseDir, e.Name()))
		if len(data) > 0 {
			s.data[e.Name()] = data
		}
	}
}

func (s *Store) Put(key string, val any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	path := filepath.Join(s.baseDir, key)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	s.data[key] = data
	return nil
}

func (s *Store) Get(key string, out any) error {
	s.mu.RLock()
	data, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	os.Remove(filepath.Join(s.baseDir, key))
	return nil
}

func (s *Store) List(prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for k := range s.data {
		if len(prefix) == 0 || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}