// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

// BadgerDB Store implements persistence for Coco sandbox state.
// Schema:
//   - sandbox:{id}          → JSON of Sandbox
//   - sandbox_index         → sorted set of sandbox IDs (keystroke indexed)
//   - checkpoint:{sid}:{cid} → checkpoint metadata
//   - replay:{sid}          → replay session metadata

type Store struct {
	mu      sync.RWMutex
	db      *badger.DB
	baseDir string
}

func New(baseDir string) (*Store, error) {
	os.MkdirAll(baseDir, 0755)

	opts := badger.Options{
		Dir:                baseDir,
		ValueDir:            baseDir,
		MemoryMapLoadingMmap: false,
		Compression:         badger.ZSTD,
		MetricsEnabled:      true,
		MaxTableSize:        64 << 20, // 64MB
		LmaxCompaction:      true,
	}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("badger open: %w", err)
	}

	return &Store{
		db:      db,
		baseDir: baseDir,
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Put stores a value under the given key.
func (s *Store) Put(key string, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// Get retrieves a value by key into out.
func (s *Store) Get(key string, out any) error {
	var data []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		data, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return fmt.Errorf("badger get: %w", err)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// Delete removes a key.
func (s *Store) Delete(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// List returns keys matching the given prefix.
func (s *Store) List(prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek([]byte(prefix)); iter.ValidForPrefix([]byte(prefix)); iter.Next() {
			key := iter.Item().Key()
			keys = append(keys, string(key))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("badger list: %w", err)
	}
	return keys, nil
}

// PutSandbox stores a sandbox with index maintenance.
func (s *Store) PutSandbox(id string, sb any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := "sandbox:" + id
	data, err := json.Marshal(sb)
	if err != nil {
		return fmt.Errorf("marshal sandbox: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(key), data); err != nil {
			return err
		}
		// Add to index
		return txn.Set([]byte("sandbox_index:"+id), []byte{})
	})
}

// GetSandbox retrieves a sandbox by ID.
func (s *Store) GetSandbox(id string, out any) error {
	return s.Get("sandbox:"+id, out)
}

// DeleteSandbox removes a sandbox and its index entry.
func (s *Store) DeleteSandbox(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete([]byte("sandbox:" + id)); err != nil {
			return err
		}
		return txn.Delete([]byte("sandbox_index:" + id))
	})
}

// ListSandboxes returns all sandbox IDs.
func (s *Store) ListSandboxes() ([]string, error) {
	return s.List("sandbox_index:")
}

// PutCheckpoint stores checkpoint metadata.
func (s *Store) PutCheckpoint(sandboxID, checkpointID string, cp any) error {
	key := fmt.Sprintf("checkpoint:%s:%s", sandboxID, checkpointID)
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// GetCheckpoint retrieves a checkpoint.
func (s *Store) GetCheckpoint(sandboxID, checkpointID string, out any) error {
	return s.Get(fmt.Sprintf("checkpoint:%s:%s", sandboxID, checkpointID), out)
}

// ListCheckpoints returns all checkpoint IDs for a sandbox.
func (s *Store) ListCheckpoints(sandboxID string) ([]string, error) {
	prefix := fmt.Sprintf("checkpoint:%s:", sandboxID)
	keys, err := s.List(prefix)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(keys))
	suffix := ":" + sandboxID + ":"
	for _, k := range keys {
		// Strip prefix and suffix to get checkpoint ID
		id := k[len(prefix):]
		ids = append(ids, id)
	}
	return ids, nil
}

// DeleteCheckpoint removes a checkpoint.
func (s *Store) DeleteCheckpoint(sandboxID, checkpointID string) error {
	return s.Delete(fmt.Sprintf("checkpoint:%s:%s", sandboxID, checkpointID))
}

// PutReplay stores replay session metadata.
func (s *Store) PutReplay(sandboxID string, rp any) error {
	key := "replay:" + sandboxID
	data, err := json.Marshal(rp)
	if err != nil {
		return fmt.Errorf("marshal replay: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// GetReplay retrieves a replay session.
func (s *Store) GetReplay(sandboxID string, out any) error {
	return s.Get("replay:"+sandboxID, out)
}

// RunGarbages collects tombstones and compacts the database.
func (s *Store) RunGarbages() error {
	return s.db.RunGarbages()
}

// Stats returns badger DB statistics.
func (s *Store) Stats() (badger.DBStats, error) {
	var stats badger.DBStats
	err := s.db.View(func(txn *badger.Txn) error {
		stats = s.db.DbStats()
		return nil
	})
	return stats, err
}

// Ensure dir exists (used by JSON store compat).
func ensureDir(dir string) {
	os.MkdirAll(dir, 0755)
}

func sandboxKey(id string) string   { return "sandbox:" + id }
func checkpointKey(sid, cid string) string { return fmt.Sprintf("checkpoint:%s:%s", sid, cid) }
func replayKey(sid string) string    { return "replay:" + sid }

// =============================================================================
// Legacy JSON file store compat — used if BadgerDB unavailable
// =============================================================================

// JSONStore is a fallback when BadgerDB cannot be opened.
type JSONStore struct {
	mu      sync.RWMutex
	baseDir string
	data    map[string][]byte
}

func newJSONStore(baseDir string) (*JSONStore, error) {
	ensureDir(baseDir)
	js := &JSONStore{
		baseDir: baseDir,
		data:    make(map[string][]byte),
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return js, nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(baseDir, e.Name()))
		if len(data) > 0 {
			js.data[e.Name()] = data
		}
	}
	return js, nil
}

func (js *JSONStore) Put(key string, val any) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := filepath.Join(js.baseDir, key)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	js.data[key] = data
	return nil
}

func (js *JSONStore) Get(key string, out any) error {
	js.mu.RLock()
	data, ok := js.data[key]
	js.mu.RUnlock()
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

func (js *JSONStore) Delete(key string) error {
	js.mu.Lock()
	defer js.mu.Unlock()
	delete(js.data, key)
	os.Remove(filepath.Join(js.baseDir, key))
	return nil
}

func (js *JSONStore) List(prefix string) ([]string, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()
	var keys []string
	for k := range js.data {
		if len(prefix) == 0 || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

var _ = log.Println // silence unused import in JSONStore fallback
