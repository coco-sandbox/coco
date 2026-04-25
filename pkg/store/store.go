// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package store provides data persistence for the Coco API server.
package store

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/coco-sandbox/coco/internal/types"
)

// Store provides data persistence using BadgerDB
type Store struct {
	db *badger.DB
}

// New creates a new Store
func New(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir)
	opts.SyncWrites = false

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the store
func (s *Store) Close() error {
	return s.db.Close()
}

// GetSandbox retrieves a sandbox by ID
func (s *Store) GetSandbox(id string) (*types.Sandbox, error) {
	key := []byte(fmt.Sprintf("sandbox:%s", id))
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	var sb types.Sandbox
	valBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valBytes, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}

// PutSandbox stores a sandbox
func (s *Store) PutSandbox(sb *types.Sandbox) error {
	key := []byte(fmt.Sprintf("sandbox:%s", sb.ID))
	val, err := json.Marshal(sb)
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

// DeleteSandbox removes a sandbox
func (s *Store) DeleteSandbox(id string) error {
	key := []byte(fmt.Sprintf("sandbox:%s", id))
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// ListSandboxes returns all sandboxes
func (s *Store) ListSandboxes() ([]*types.Sandbox, error) {
	sandboxes := make([]*types.Sandbox, 0)

	_, err := s.db.View(func(txn *badger.Txn) error {
		prefix := []byte("sandbox:")
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var sb types.Sandbox
			valBytes, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(valBytes, &sb); err != nil {
				return err
			}
			sandboxes = append(sandboxes, &sb)
		}
		return nil
	})

	return sandboxes, err
}

// GetCheckpoint retrieves a checkpoint
func (s *Store) GetCheckpoint(sandboxID, checkpointID string) (*types.Checkpoint, error) {
	key := []byte(fmt.Sprintf("checkpoint:%s:%s", sandboxID, checkpointID))
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	var cp types.Checkpoint
	valBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valBytes, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

// PutCheckpoint stores a checkpoint
func (s *Store) PutCheckpoint(cp *types.Checkpoint) error {
	key := []byte(fmt.Sprintf("checkpoint:%s:%s", cp.SandboxID, cp.ID))
	val, err := json.Marshal(cp)
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}
