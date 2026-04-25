// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// =============================================================================
// Store Types
// =============================================================================

type Store struct {
	db *badger.DB
}

// Index keys
const (
	indexSandboxByTenant    = "sandbox:tenant:%s"        // tenant -> []sandboxID
	indexSandboxByState     = "sandbox:state:%s"         // state -> []sandboxID
	indexSandboxByCreated   = "sandbox:created:%d:%s"    // timestamp -> sandboxID
	indexCheckpointBySandbox = "checkpoint:sandbox:%s"  // sandboxID -> []checkpointID
)

// =============================================================================
// Store Initialization
// =============================================================================

func newStore(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir)
	opts.SyncWrites = false           // Performance tuning

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DropAll() error {
	return s.db.DropAll()
}

// =============================================================================
// Sandbox Store Operations
// =============================================================================

func (s *Store) GetSandbox(id string) (*Sandbox, error) {
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

	var sb Sandbox
	valBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valBytes, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}

func (s *Store) PutSandbox(sb *Sandbox) error {
	key := []byte(fmt.Sprintf("sandbox:%s", sb.ID))
	val, err := json.Marshal(sb)
	if err != nil {
		return err
	}

	// Update indexes in a write transaction
	txn := s.db.NewTransaction(true)
	defer txn.Commit()

	// Put sandbox data
	if err := txn.Set(key, val); err != nil {
		txn.Discard()
		return err
	}

	// Update tenant index
	tenantKey := fmt.Sprintf(indexSandboxByTenant, "default") // TODO: support multi-tenant
	if err := txn.Set([]byte(tenantKey), appendOrCreate(txOrGet(txn, tenantKey), sb.ID)); err != nil {
		txn.Discard()
		return err
	}

	// Update state index
	stateKey := fmt.Sprintf(indexSandboxByState, sb.State.String())
	if err := txn.Set([]byte(stateKey), appendOrCreate(txOrGet(txn, stateKey), sb.ID)); err != nil {
		txn.Discard()
		return err
	}

	// Update created_at index
	createdKey := fmt.Sprintf(indexSandboxByCreated, sb.CreatedAt.UnixNano(), sb.ID)
	if err := txn.Set([]byte(createdKey), []byte(sb.ID)); err != nil {
		txn.Discard()
		return err
	}

	return txn.Commit()
}

func (s *Store) DeleteSandbox(id string) error {
	sb, err := s.GetSandbox(id)
	if err != nil {
		return err
	}
	if sb == nil {
		return nil
	}

	txn := s.db.NewTransaction(true)
	defer txn.Commit()

	// Delete sandbox data
	if err := txn.Delete([]byte(fmt.Sprintf("sandbox:%s", id))); err != nil {
		txn.Discard()
		return err
	}

	// Remove from tenant index
	tenantKey := fmt.Sprintf(indexSandboxByTenant, "default")
	if err := txn.Set([]byte(tenantKey), removeFromList(txOrGet(txn, tenantKey), id)); err != nil {
		txn.Discard()
		return err
	}

	// Remove from state index
	stateKey := fmt.Sprintf(indexSandboxByState, sb.State.String())
	if err := txn.Set([]byte(stateKey), removeFromList(txOrGet(txn, stateKey), id)); err != nil {
		txn.Discard()
		return err
	}

	return txn.Commit()
}

func (s *Store) ListSandboxesByState(state SandboxState) ([]string, error) {
	key := fmt.Sprintf(indexSandboxByState, state.String())
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get([]byte(key))
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return []string{}, nil
		}
		return nil, err
	}
	valBytes, _ := item.ValueCopy(nil)
	return parseIDList(valBytes), nil
}

func (s *Store) ListSandboxesByTenant(tenantID string) ([]string, error) {
	key := fmt.Sprintf(indexSandboxByTenant, tenantID)
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get([]byte(key))
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return []string{}, nil
		}
		return nil, err
	}
	valBytes, _ := item.ValueCopy(nil)
	return parseIDList(valBytes), nil
}

// =============================================================================
// Checkpoint Store Operations
// =============================================================================

func (s *Store) GetCheckpoint(sandboxID, checkpointID string) (*Checkpoint, error) {
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

	var cp Checkpoint
	valBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valBytes, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

func (s *Store) PutCheckpoint(cp *Checkpoint) error {
	key := []byte(fmt.Sprintf("checkpoint:%s:%s", cp.SandboxID, cp.ID))
	val, err := json.Marshal(cp)
	if err != nil {
		return err
	}

	txn := s.db.NewTransaction(true)
	defer txn.Commit()

	if err := txn.Set(key, val); err != nil {
		txn.Discard()
		return err
	}

	// Update sandbox checkpoint index
	indexKey := fmt.Sprintf(indexCheckpointBySandbox, cp.SandboxID)
	if err := txn.Set([]byte(indexKey), appendOrCreate(txOrGet(txn, indexKey), cp.ID)); err != nil {
		txn.Discard()
		return err
	}

	return txn.Commit()
}

func (s *Store) DeleteCheckpoint(sandboxID, checkpointID string) error {
	txn := s.db.NewTransaction(true)
	defer txn.Commit()

	// Delete checkpoint data
	key := []byte(fmt.Sprintf("checkpoint:%s:%s", sandboxID, checkpointID))
	if err := txn.Delete(key); err != nil {
		txn.Discard()
		return err
	}

	// Remove from index
	indexKey := fmt.Sprintf(indexCheckpointBySandbox, sandboxID)
	if err := txn.Set([]byte(indexKey), removeFromList(txOrGet(txn, indexKey), checkpointID)); err != nil {
		txn.Discard()
		return err
	}

	return txn.Commit()
}

func (s *Store) ListCheckpointsBySandbox(sandboxID string) ([]*Checkpoint, error) {
	indexKey := fmt.Sprintf(indexCheckpointBySandbox, sandboxID)
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get([]byte(indexKey))
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return []*Checkpoint{}, nil
		}
		return nil, err
	}
	valBytes, _ := item.ValueCopy(nil)

	var checkpoints []*Checkpoint
	for _, id := range parseIDList(valBytes) {
		cp, err := s.GetCheckpoint(sandboxID, id)
		if err != nil {
			return nil, err
		}
		if cp != nil {
			checkpoints = append(checkpoints, cp)
		}
	}
	return checkpoints, nil
}

// =============================================================================
// Node Store Operations
// =============================================================================

type Node struct {
	ID        string    `json:"id"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Capacity  int       `json:"capacity"`
	State     string    `json:"state"` // healthy, unhealthy, joining, leaving
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) GetNode(id string) (*Node, error) {
	key := []byte(fmt.Sprintf("node:%s", id))
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	var node Node
	valBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valBytes, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *Store) PutNode(node *Node) error {
	key := []byte(fmt.Sprintf("node:%s", node.ID))
	val, err := json.Marshal(node)
	if err != nil {
		return err
	}
	txn := s.db.NewTransaction(true)
	if err := txn.Set(key, val); err != nil {
		txn.Discard()
		return err
	}
	return txn.Commit()
}

func (s *Store) DeleteNode(id string) error {
	key := []byte(fmt.Sprintf("node:%s", id))
	txn := s.db.NewTransaction(true)
	if err := txn.Delete(key); err != nil {
		txn.Discard()
		return err
	}
	return txn.Commit()
}

func (s *Store) ListNodes() ([]*Node, error) {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	var nodes []*Node
	prefix := []byte("node:")
	for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
		var node Node
		item := iter.Item()
		valBytes, err := item.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(valBytes, &node); err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

// =============================================================================
// Template Store Operations
// =============================================================================

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImagePath   string `json:"image_path"`
	MemoryMB    int    `json:"memory_mb"`
	VCPUs       int    `json:"vcpus"`
	Default     bool   `json:"default"`
}

func (s *Store) GetTemplate(id string) (*Template, error) {
	key := []byte(fmt.Sprintf("template:%s", id))
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	var tmpl Template
	valBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valBytes, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

func (s *Store) PutTemplate(tmpl *Template) error {
	key := []byte(fmt.Sprintf("template:%s", tmpl.ID))
	val, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}
	txn := s.db.NewTransaction(true)
	if err := txn.Set(key, val); err != nil {
		txn.Discard()
		return err
	}
	return txn.Commit()
}

func (s *Store) DeleteTemplate(id string) error {
	key := []byte(fmt.Sprintf("template:%s", id))
	txn := s.db.NewTransaction(true)
	if err := txn.Delete(key); err != nil {
		txn.Discard()
		return err
	}
	return txn.Commit()
}

func (s *Store) ListTemplates() ([]*Template, error) {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	var templates []*Template
	prefix := []byte("template:")
	for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
		var tmpl Template
		item := iter.Item()
		valBytes, err := item.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(valBytes, &tmpl); err != nil {
			return nil, err
		}
		templates = append(templates, &tmpl)
	}
	return templates, nil
}

func (s *Store) GetDefaultTemplate() (*Template, error) {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	prefix := []byte("template:")
	for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
		var tmpl Template
		item := iter.Item()
		valBytes, err := item.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(valBytes, &tmpl); err != nil {
			return nil, err
		}
		if tmpl.Default {
			return &tmpl, nil
		}
	}
	return nil, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func appendOrCreate(data []byte, id string) []byte {
	if len(data) == 0 {
		return []byte(id + ",")
	}
	return append(data, id...)
}

func removeFromList(data []byte, id string) []byte {
	list := parseIDList(data)
	for i, v := range list {
		if v == id {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
	return joinIDs(list)
}

func parseIDList(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	parts := strings.Split(string(data), ",")
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func joinIDs(ids []string) []byte {
	if len(ids) == 0 {
		return nil
	}
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(id)
	}
	b.WriteByte(',')
	return []byte(b.String())
}

func txOrGet(txn *badger.Txn, key string) []byte {
	item, err := txn.Get([]byte(key))
	if err != nil {
		return nil
	}
	val, _ := item.ValueCopy(nil)
	return val
}