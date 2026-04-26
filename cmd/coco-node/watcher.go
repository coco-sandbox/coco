package main

import (
	"context"
	"log"
	"sync"
	"time"
)

type NodeResources struct {
	CPU       float64
	Memory    int64
	Disk      int64
	Sandboxes int
}

type WatchEvent struct {
	Type      string
	Resources *NodeResources
}

type DiscoveryClient interface {
	GetNodeResources(ctx context.Context, nodeID string) (*NodeResources, error)
}

type Watcher struct {
	mu           sync.RWMutex
	watchers     map[chan *WatchEvent]bool
	nodeID       string
	discovery    DiscoveryClient
	pollInterval time.Duration
}

func NewWatcher(nodeID string, discovery DiscoveryClient) *Watcher {
	return &Watcher{
		watchers:     make(map[chan *WatchEvent]bool),
		nodeID:       nodeID,
		discovery:    discovery,
		pollInterval: 5 * time.Second,
	}
}

func (w *Watcher) Subscribe() <-chan *WatchEvent {
	ch := make(chan *WatchEvent, 10)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.watchers[ch] = true
	return ch
}

func (w *Watcher) Unsubscribe(ch <-chan *WatchEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for key := range w.watchers {
		if key == ch {
			delete(w.watchers, key)
			break
		}
	}
}

func (w *Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkResources(ctx)
		}
	}
}

func (w *Watcher) checkResources(ctx context.Context) {
	resources, err := w.discovery.GetNodeResources(ctx, w.nodeID)
	if err != nil {
		log.Printf("failed to get node resources: %v", err)
		return
	}

	event := &WatchEvent{
		Type:      "update",
		Resources: resources,
	}

	w.notify(event)
}

func (w *Watcher) notify(event *WatchEvent) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for ch := range w.watchers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (w *Watcher) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for ch := range w.watchers {
		close(ch)
	}
	w.watchers = make(map[chan *WatchEvent]bool)
}
