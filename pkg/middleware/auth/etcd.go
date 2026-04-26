// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3"
)

const (
	etcdAPIKeyPrefix = "/coco/api-keys/"
	etcdAPIKeyTTL    = 10 * time.Second
)

type etcdStore struct {
	client *clientv3.Client
	prefix string
}

func NewEtcdStore(endpoints []string) (*etcdStore, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &etcdStore{
		client: cli,
		prefix: etcdAPIKeyPrefix,
	}, nil
}

func (s *etcdStore) CreateKey(ctx context.Context, key *APIKey) (string, error) {
	data, err := json.Marshal(key)
	if err != nil {
		return "", fmt.Errorf("failed to marshal key: %w", err)
	}

	lease, err := s.client.Grant(ctx, int64(etcdAPIKeyTTL.Seconds()))
	if err != nil {
		return "", fmt.Errorf("failed to create lease: %w", err)
	}

	_, err = s.client.Put(ctx, s.prefix+key.ID, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		return "", fmt.Errorf("failed to put key: %w", err)
	}

	return key.ID, nil
}

func (s *etcdStore) GetKey(ctx context.Context, id string) (*APIKey, error) {
	resp, err := s.client.Get(ctx, s.prefix+id)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrInvalidKey
	}

	var key APIKey
	if err := json.Unmarshal(resp.Kvs[0].Value, &key); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key: %w", err)
	}

	return &key, nil
}

func (s *etcdStore) GetKeyByHash(ctx context.Context, hash string) (*APIKey, error) {
	resp, err := s.client.Get(ctx, s.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	for _, kv := range resp.Kvs {
		var key APIKey
		if err := json.Unmarshal(kv.Value, &key); err != nil {
			continue
		}
		if key.KeyHash == hash {
			return &key, nil
		}
	}

	return nil, ErrInvalidKey
}

func (s *etcdStore) ListKeys(ctx context.Context) ([]*APIKey, error) {
	resp, err := s.client.Get(ctx, s.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	keys := make([]*APIKey, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var key APIKey
		if err := json.Unmarshal(kv.Value, &key); err != nil {
			continue
		}
		keys = append(keys, &key)
	}

	return keys, nil
}

func (s *etcdStore) DeleteKey(ctx context.Context, id string) error {
	_, err := s.client.Delete(ctx, s.prefix+id)
	return err
}

func (s *etcdStore) ValidateKey(ctx context.Context, rawKey string) (*APIKey, error) {
	hash := HashKey(rawKey)
	return s.GetKeyByHash(ctx, hash)
}

func (s *etcdStore) Close() error {
	return s.client.Close()
}
