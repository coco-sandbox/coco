package store

import (
	"context"
	"time"
)

type Store struct {
	backend Backend
}

type Backend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Watch(ctx context.Context, prefix string) (<-chan []string, error)
}

func New(backend Backend) *Store {
	return &Store{
		backend: backend,
	}
}

func (s *Store) Get(key string) ([]byte, error) {
	return s.backend.Get(context.Background(), key)
}

func (s *Store) Put(key string, value []byte) error {
	return s.backend.Put(context.Background(), key, value)
}

func (s *Store) Delete(key string) error {
	return s.backend.Delete(context.Background(), key)
}

func (s *Store) List(prefix string) ([]string, error) {
	return s.backend.List(context.Background(), prefix)
}

func (s *Store) Watch(prefix string) (<-chan []string, error) {
	return s.backend.Watch(context.Background(), prefix)
}

type KeyValue struct {
	Key      string
	Value    []byte
	Revision int64
	Created  time.Time
	Modified time.Time
}

type Iterator interface {
	Next() bool
	Item() *KeyValue
	Close() error
}

type Query struct {
	Prefix    string
	Limit     int
	Offset    string
	Reverse   bool
}

func (s *Store) Query(q Query) (Iterator, error) {
	keys, err := s.List(q.Prefix)
	if err != nil {
		return nil, err
	}

	start := 0
	if q.Offset != "" {
		for i, k := range keys {
			if k == q.Offset {
				start = i + 1
				break
			}
		}
	}

	end := len(keys)
	if q.Limit > 0 {
		end = start + q.Limit
		if end > len(keys) {
			end = len(keys)
		}
	}

	if q.Reverse {
		keys = reverseSlice(keys[start:end])
	} else {
		keys = keys[start:end]
	}

	return &kvIterator{keys: keys, store: s}, nil
}

type kvIterator struct {
	keys  []string
	store *Store
	index int
}

func (it *kvIterator) Next() bool {
	return it.index < len(it.keys)
}

func (it *kvIterator) Item() *KeyValue {
	if !it.Next() {
		return nil
	}
	key := it.keys[it.index]
	val, _ := it.store.Get(key)
	it.index++
	return &KeyValue{Key: key, Value: val}
}

func (it *kvIterator) Close() error {
	return nil
}

func reverseSlice[T any](s []T) []T {
	result := make([]T, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}
