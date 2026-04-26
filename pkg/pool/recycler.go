package pool

import (
	"context"
	"sync"
	"time"
)

type Recycler[T any] struct {
	mu       sync.Mutex
	items    []T
	callback func(T) error
	interval time.Duration
}

func NewRecycler[T any](callback func(T) error, interval time.Duration) *Recycler[T] {
	return &Recycler[T]{
		callback: callback,
		interval: interval,
		items:    make([]T, 0),
	}
}

func (r *Recycler[T]) Add(item T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, item)
}

func (r *Recycler[T]) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.recycle()
		}
	}
}

func (r *Recycler[T]) recycle() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range r.items {
		if r.callback != nil {
			_ = r.callback(item)
		}
	}
	r.items = r.items[:0]
}

func (r *Recycler[T]) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.items)
}
