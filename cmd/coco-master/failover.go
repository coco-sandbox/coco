package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/scheduler"
)

type FailoverManager struct {
	mu           sync.RWMutex
	nodes        map[string]*FailedNode
	sandboxes    map[string]*FailedSandbox
	scheduler    *scheduler.Scheduler
	checkpoint   CheckpointManager
	onNodeFail   func(nodeID string)
	onSandboxFail func(sandboxID, nodeID string)
	checkInterval time.Duration
	maxRetries   int
}

type FailedNode struct {
	ID        string
	FailedAt  time.Time
	Retries   int
}

type FailedSandbox struct {
	ID           string
	NodeID       string
	FailedAt     time.Time
	HasCheckpoint bool
	Retries      int
}

type CheckpointManager interface {
	HasCheckpoint(sandboxID string) (bool, error)
	Restore(ctx context.Context, sandboxID, nodeID string) error
}

func NewFailoverManager(sched *scheduler.Scheduler, checkpoint CheckpointManager) *FailoverManager {
	return &FailoverManager{
		nodes:         make(map[string]*FailedNode),
		sandboxes:    make(map[string]*FailedSandbox),
		scheduler:    sched,
		checkpoint:   checkpoint,
		checkInterval: 10 * time.Second,
		maxRetries:   3,
	}
}

func (f *FailoverManager) Start(ctx context.Context) {
	ticker := time.NewTicker(f.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.processFailures(ctx)
		}
	}
}

func (f *FailoverManager) RegisterNodeFailure(nodeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.nodes[nodeID]; !ok {
		f.nodes[nodeID] = &FailedNode{
			ID:       nodeID,
			FailedAt: time.Now(),
			Retries:  0,
		}
		log.Printf("Node %s marked as failed", nodeID)
	}

	if f.onNodeFail != nil {
		f.onNodeFail(nodeID)
	}
}

func (f *FailoverManager) RegisterSandboxFailure(sandboxID, nodeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	hasCheckpoint := false
	if f.checkpoint != nil {
		if ok, err := f.checkpoint.HasCheckpoint(sandboxID); err == nil {
			hasCheckpoint = ok
		}
	}

	f.sandboxes[sandboxID] = &FailedSandbox{
		ID:            sandboxID,
		NodeID:        nodeID,
		FailedAt:      time.Now(),
		HasCheckpoint: hasCheckpoint,
		Retries:       0,
	}
	log.Printf("Sandbox %s on node %s marked as failed (checkpoint: %v)", sandboxID, nodeID, hasCheckpoint)

	if f.onSandboxFail != nil {
		f.onSandboxFail(sandboxID, nodeID)
	}
}

func (f *FailoverManager) processFailures(ctx context.Context) {
	f.processNodeFailures(ctx)
	f.processSandboxFailures(ctx)
}

func (f *FailoverManager) processNodeFailures(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for nodeID, node := range f.nodes {
		if node.Retries >= f.maxRetries {
			log.Printf("Node %s exceeded max retries, removing from failover", nodeID)
			delete(f.nodes, nodeID)
			continue
		}

		log.Printf("Processing failed node %s (retry %d/%d)", nodeID, node.Retries+1, f.maxRetries)
		node.Retries++

		for sandboxID, sandbox := range f.sandboxes {
			if sandbox.NodeID == nodeID {
				if sandbox.HasCheckpoint {
					if err := f.restoreSandbox(ctx, sandboxID); err != nil {
						log.Printf("Failed to restore sandbox %s: %v", sandboxID, err)
					} else {
						delete(f.sandboxes, sandboxID)
					}
				} else {
					log.Printf("Sandbox %s has no checkpoint, marking as lost", sandboxID)
					delete(f.sandboxes, sandboxID)
				}
			}
		}
	}
}

func (f *FailoverManager) processSandboxFailures(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for sandboxID, sandbox := range f.sandboxes {
		if sandbox.Retries >= f.maxRetries {
			log.Printf("Sandbox %s exceeded max retries, giving up", sandboxID)
			delete(f.sandboxes, sandboxID)
			continue
		}

		if sandbox.HasCheckpoint {
			log.Printf("Processing failed sandbox %s (retry %d/%d)", sandboxID, sandbox.Retries+1, f.maxRetries)
			sandbox.Retries++

			if err := f.restoreSandbox(ctx, sandboxID); err != nil {
				log.Printf("Failed to restore sandbox %s: %v", sandboxID, err)
			} else {
				delete(f.sandboxes, sandboxID)
			}
		}
	}
}

func (f *FailoverManager) restoreSandbox(ctx context.Context, sandboxID string) error {
	if f.checkpoint == nil {
		return fmt.Errorf("no checkpoint manager available")
	}

	node, err := f.scheduler.Schedule(scheduler.StrategyLeastLoaded)
	if err != nil {
		return fmt.Errorf("no nodes available for restore: %w", err)
	}

	log.Printf("Restoring sandbox %s on node %s", sandboxID, node.ID)
	return f.checkpoint.Restore(ctx, sandboxID, node.ID)
}

func (f *FailoverManager) GetFailedNodes() map[string]*FailedNode {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]*FailedNode)
	for k, v := range f.nodes {
		result[k] = v
	}
	return result
}

func (f *FailoverManager) GetFailedSandboxes() map[string]*FailedSandbox {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]*FailedSandbox)
	for k, v := range f.sandboxes {
		result[k] = v
	}
	return result
}

func (f *FailoverManager) SetNodeFailureHandler(handler func(nodeID string)) {
	f.onNodeFail = handler
}

func (f *FailoverManager) SetSandboxFailureHandler(handler func(sandboxID, nodeID string)) {
	f.onSandboxFail = handler
}
