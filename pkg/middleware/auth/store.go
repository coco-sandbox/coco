// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package auth

import (
	"context"
	"sync"
)

type InMemoryStore struct {
	mu    sync.RWMutex
	keys  map[string]*APIKey
	hashes map[string]string
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		keys:  make(map[string]*APIKey),
		hashes: make(map[string]string),
	}
}

func (s *InMemoryStore) CreateKey(ctx context.Context, key *APIKey) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[key.ID] = key
	s.hashes[key.KeyHash] = key.ID

	return key.ID, nil
}

func (s *InMemoryStore) GetKey(ctx context.Context, id string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[id]
	if !ok {
		return nil, ErrInvalidKey
	}

	return key, nil
}

func (s *InMemoryStore) GetKeyByHash(ctx context.Context, hash string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.hashes[hash]
	if !ok {
		return nil, ErrInvalidKey
	}

	key, ok := s.keys[id]
	if !ok {
		return nil, ErrInvalidKey
	}

	return key, nil
}

func (s *InMemoryStore) ListKeys(ctx context.Context) ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*APIKey, 0, len(s.keys))
	for _, key := range s.keys {
		keys = append(keys, key)
	}

	return keys, nil
}

func (s *InMemoryStore) DeleteKey(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.keys[id]
	if !ok {
		return nil
	}

	delete(s.hashes, key.KeyHash)
	delete(s.keys, id)

	return nil
}

func (s *InMemoryStore) ValidateKey(ctx context.Context, rawKey string) (*APIKey, error) {
	hash := HashKey(rawKey)
	return s.GetKeyByHash(ctx, hash)
}

func (s *InMemoryStore) AddKey(key *APIKey) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[key.ID] = key
	s.hashes[key.KeyHash] = key.ID
}
