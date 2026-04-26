package pool

import (
	"sync"
)

type Watcher[T any] struct {
	mu       sync.RWMutex
	watchers map[chan T]bool
}

func NewWatcher[T any]() *Watcher[T] {
	return &Watcher[T]{
		watchers: make(map[chan T]bool),
	}
}

func (w *Watcher[T]) Subscribe() chan T {
	ch := make(chan T, 10)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.watchers[ch] = true
	return ch
}

func (w *Watcher[T]) Unsubscribe(ch chan T) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.watchers, ch)
}

func (w *Watcher[T]) Notify(item T) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for ch := range w.watchers {
		select {
		case ch <- item:
		default:
		}
	}
}

func (w *Watcher[T]) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for ch := range w.watchers {
		close(ch)
	}
	w.watchers = make(map[chan T]bool)
}

type PoolWatcher struct {
	watcher *Watcher[*PoolEvent]
}

type PoolEvent struct {
	Type    string
	ItemID  string
	Payload interface{}
}

func NewPoolWatcher() *PoolWatcher {
	return &PoolWatcher{
		watcher: NewWatcher[*PoolEvent](),
	}
}

func (pw *PoolWatcher) Subscribe() <-chan *PoolEvent {
	return pw.watcher.Subscribe()
}

func (pw *PoolWatcher) Notify(event *PoolEvent) {
	pw.watcher.Notify(event)
}

func (pw *PoolWatcher) Close() {
	pw.watcher.Close()
}
