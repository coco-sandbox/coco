package store

import (
	"context"
	"fmt"
)

type Transaction interface {
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
	Delete(key string) error
	Commit() error
	Rollback() error
}

type storeTransaction struct {
	store  *Store
	ops    []operation
	cached map[string][]byte
}

type operation struct {
	op     string
	key    string
	value  []byte
}

func (s *Store) BeginTransaction(ctx context.Context) (Transaction, error) {
	return &storeTransaction{
		store:  s,
		ops:    make([]operation, 0),
		cached: make(map[string][]byte),
	}, nil
}

func (t *storeTransaction) Get(key string) ([]byte, error) {
	if val, ok := t.cached[key]; ok {
		return val, nil
	}
	return t.store.Get(key)
}

func (t *storeTransaction) Put(key string, value []byte) error {
	t.ops = append(t.ops, operation{op: "put", key: key, value: value})
	t.cached[key] = value
	return nil
}

func (t *storeTransaction) Delete(key string) error {
	t.ops = append(t.ops, operation{op: "delete", key: key})
	delete(t.cached, key)
	return nil
}

func (t *storeTransaction) Commit() error {
	for _, op := range t.ops {
		switch op.op {
		case "put":
			if err := t.store.Put(op.key, op.value); err != nil {
				return fmt.Errorf("failed to put %s: %w", op.key, err)
			}
		case "delete":
			if err := t.store.Delete(op.key); err != nil {
				return fmt.Errorf("failed to delete %s: %w", op.key, err)
			}
		}
	}
	t.ops = nil
	return nil
}

func (t *storeTransaction) Rollback() error {
	t.ops = nil
	t.cached = make(map[string][]byte)
	return nil
}
