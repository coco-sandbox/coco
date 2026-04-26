// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package cluster provides cluster management for the Coco API server.
package cluster

import (
	"fmt"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/types"
)

// Manager manages cluster membership and health
type Manager struct {
	mu      sync.RWMutex
	nodes   map[string]*types.ClusterNode
	selfID  string
	version string
	stopCh  chan struct{}
}

// NewManager creates a new cluster manager
func NewManager(selfID, version string) *Manager {
	return &Manager{
		nodes:   make(map[string]*types.ClusterNode),
		selfID:  selfID,
		version: version,
		stopCh:  make(chan struct{}),
	}
}

// Start begins background tasks
func (m *Manager) Start() {
	m.mu.Lock()
	m.nodes[m.selfID] = &types.ClusterNode{
		ID:        m.selfID,
		State:     "alive",
		Role:      "primary",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
		Priority:  100,
		Version:   m.version,
	}
	m.mu.Unlock()
}

// Stop shuts down the cluster manager
func (m *Manager) Stop() {
	close(m.stopCh)
}

// RegisterNode adds a new node to the cluster
func (m *Manager) RegisterNode(node *types.ClusterNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already registered", node.ID)
	}

	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	node.LastSeen = time.Now()
	m.nodes[node.ID] = node

	return nil
}

// UnregisterNode removes a node from the cluster
func (m *Manager) UnregisterNode(nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(m.nodes, nodeID)
	return nil
}

// ListNodes returns all known nodes
func (m *Manager) ListNodes() []*types.ClusterNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*types.ClusterNode, 0, len(m.nodes))
	for _, n := range m.nodes {
		result = append(result, n)
	}
	return result
}

// GetNode returns a specific node
func (m *Manager) GetNode(nodeID string) (*types.ClusterNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}
	return node, nil
}

// GetAliveNodes returns all nodes in "alive" state
func (m *Manager) GetAliveNodes() []*types.ClusterNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*types.ClusterNode, 0)
	for _, n := range m.nodes {
		if n.State == "alive" {
			result = append(result, n)
		}
	}
	return result
}

// IsLeader returns true if this node is the cluster leader
func (m *Manager) IsLeader() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nodes[m.selfID] != nil && m.nodes[m.selfID].Role == "primary"
}

// ShardKey generates a consistent hash key for a sandbox ID
func ShardKey(sandboxID string) uint32 {
	h := uint32(2166136261)
	for _, c := range sandboxID {
		h ^= uint32(c)
		h *= 16777619
	}
	return h
}
