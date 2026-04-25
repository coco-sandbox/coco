// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	baseDir string
	mu      sync.RWMutex
	index   map[string]*Template
}

func NewStore(baseDir string) (*Store, error) {
	s := &Store{baseDir: baseDir, index: make(map[string]*Template)}
	if err := s.loadIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Put(tpl *Template) error {
	s.mu.Lock()
	s.index[tpl.ID] = tpl
	s.mu.Unlock()

	metaPath := filepath.Join(s.baseDir, tpl.ID, "meta.json")
	os.MkdirAll(filepath.Dir(metaPath), 0755)
	data, _ := json.Marshal(tpl)
	return os.WriteFile(metaPath, data, 0644)
}

func (s *Store) Get(id string) (*Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tpl, ok := s.index[id]; ok {
		return tpl, nil
	}
	return nil, ErrTemplateNotFound
}

func (s *Store) List() ([]*Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Template, 0, len(s.index))
	for _, t := range s.index {
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.index, id)
	return os.RemoveAll(filepath.Join(s.baseDir, id))
}

func (s *Store) loadIndex() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			metaPath := filepath.Join(s.baseDir, e.Name(), "meta.json")
			if data, err := os.ReadFile(metaPath); err == nil {
				var tpl Template
				if json.Unmarshal(data, &tpl) == nil {
					s.index[tpl.ID] = &tpl
				}
			}
		}
	}
	return nil
}