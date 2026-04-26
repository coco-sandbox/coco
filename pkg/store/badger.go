// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package store

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/coco-sandbox/coco/pkg/types"
	"github.com/dgraph-io/badger/v4"
)

// BadgerStore implements persistent sandbox storage using BadgerDB
type BadgerStore struct {
	db *badger.DB
	mu sync.RWMutex
}

// NewBadgerStore creates a new BadgerDB-backed store
func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // suppress logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger: %w", err)
	}

	return &BadgerStore{db: db}, nil
}

// PutSandbox persists a sandbox
func (s *BadgerStore) PutSandbox(sb *types.Sandbox) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte("sandbox:" + sb.ID)
		val, err := json.Marshal(sb)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
}

// GetSandbox retrieves a sandbox by ID
func (s *BadgerStore) GetSandbox(id string) (*types.Sandbox, error) {
	var sb types.Sandbox
	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte("sandbox:" + id)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &sb)
		})
	})
	if err != nil {
		return nil, ErrNotFound
	}
	return &sb, nil
}

// ListSandboxes returns all sandboxes
func (s *BadgerStore) ListSandboxes() ([]*types.Sandbox, error) {
	var sandboxes []*types.Sandbox
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("sandbox:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var sb types.Sandbox
			if err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &sb)
			}); err == nil {
				sandboxes = append(sandboxes, &sb)
			}
		}
		return nil
	})
	return sandboxes, err
}

// DeleteSandbox removes a sandbox
func (s *BadgerStore) DeleteSandbox(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("sandbox:" + id))
	})
}

// Close closes the BadgerDB
func (s *BadgerStore) Close() error {
	return s.db.Close()
}

// ErrNotFound is returned when a sandbox is not found
var ErrNotFound = fmt.Errorf("sandbox not found")
