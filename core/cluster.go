// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ClusterNode represents a node in the cluster
type ClusterNode struct {
	ID        string    `json:"id"`
	Addr      string    `json:"addr"`
	State     string    `json:"state"` // "alive", "dead", "draining"
	Role      string    `json:"role"`  // "primary", "secondary"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastSeen  time.Time `json:"last_seen"`
	Priority  int       `json:"priority"` // For leader election
	Version   string    `json:"version"`
}

// ClusterManager manages node membership and health
type ClusterManager struct {
	nodes    map[string]*ClusterNode
	mu       sync.RWMutex
	selfID   string
	version  string
	stopCh   chan struct{}
	leaderID string
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(selfID, version string) *ClusterManager {
	return &ClusterManager{
		nodes:   make(map[string]*ClusterNode),
		selfID:  selfID,
		version: version,
		stopCh:  make(chan struct{}),
	}
}

// Start begins background tasks (heartbeat, health checks)
func (cm *ClusterManager) Start() {
	log.Printf("[cluster] Starting cluster manager for node %s", cm.selfID)

	// Register self
	cm.mu.Lock()
	cm.nodes[cm.selfID] = &ClusterNode{
		ID:        cm.selfID,
		State:     "alive",
		Role:      "primary", // Single node is always primary
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
		Priority:  100,
		Version:   cm.version,
	}
	cm.leaderID = cm.selfID
	cm.mu.Unlock()
}

// Stop shuts down the cluster manager
func (cm *ClusterManager) Stop() {
	close(cm.stopCh)
}

// RegisterNode adds a new node to the cluster
func (cm *ClusterManager) RegisterNode(node *ClusterNode) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already registered", node.ID)
	}

	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	node.LastSeen = time.Now()
	cm.nodes[node.ID] = node

	log.Printf("[cluster] Registered node %s (addr: %s, role: %s)", node.ID, node.Addr, node.Role)
	return nil
}

// UnregisterNode removes a node from the cluster
func (cm *ClusterManager) UnregisterNode(nodeID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(cm.nodes, nodeID)
	log.Printf("[cluster] Unregistered node %s", nodeID)
	return nil
}

// UpdateNodeState updates a node's state
func (cm *ClusterManager) UpdateNodeState(nodeID, state string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	node, exists := cm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.State = state
	node.UpdatedAt = time.Now()
	return nil
}

// RefreshNode updates the LastSeen timestamp for a node
func (cm *ClusterManager) RefreshNode(nodeID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	node, exists := cm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.LastSeen = time.Now()
	return nil
}

// ListNodes returns all known nodes
func (cm *ClusterManager) ListNodes() []*ClusterNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*ClusterNode, 0, len(cm.nodes))
	for _, n := range cm.nodes {
		result = append(result, n)
	}
	return result
}

// GetNode returns a specific node
func (cm *ClusterManager) GetNode(nodeID string) (*ClusterNode, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	node, exists := cm.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}
	return node, nil
}

// IsLeader returns true if this node is the cluster leader
func (cm *ClusterManager) IsLeader() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.leaderID == cm.selfID
}

// GetLeader returns the current cluster leader
func (cm *ClusterManager) GetLeader() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.leaderID
}

// ElectLeader performs leader election based on priority
func (cm *ClusterManager) ElectLeader() string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var best *ClusterNode
	for _, n := range cm.nodes {
		if n.State != "alive" {
			continue
		}
		if best == nil || n.Priority > best.Priority {
			best = n
		}
	}

	if best != nil {
		cm.leaderID = best.ID
		log.Printf("[cluster] Leader elected: %s", cm.leaderID)
	}

	return cm.leaderID
}

// GetAliveNodes returns all nodes in "alive" state
func (cm *ClusterManager) GetAliveNodes() []*ClusterNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*ClusterNode, 0)
	for _, n := range cm.nodes {
		if n.State == "alive" {
			result = append(result, n)
		}
	}
	return result
}

// HandleClusterNodes handles /cluster/nodes endpoint
func handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		handleClusterNodeList(w, r)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

func handleClusterNodeList(w http.ResponseWriter, r *http.Request) {
	nodes := clusterManager.ListNodes()
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes":       nodes,
		"total_count": len(nodes),
		"self_id":     clusterManager.selfID,
		"leader_id":   clusterManager.GetLeader(),
	})
}

// HandleClusterNodeByID handles /cluster/nodes/{id} endpoint
func handleClusterNodeByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/cluster/nodes/"):]
	id = strings.SplitN(id, "?", 2)[0]

	node, err := clusterManager.GetNode(id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Node %s not found", id))
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"node": node})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleClusterHealth handles /cluster/health endpoint
func handleClusterHealth(w http.ResponseWriter, r *http.Request) {
	aliveNodes := clusterManager.GetAliveNodes()

	health := map[string]any{
		"healthy":     len(aliveNodes) > 0,
		"leader_id":    clusterManager.GetLeader(),
		"is_leader":    clusterManager.IsLeader(),
		"alive_nodes":  len(aliveNodes),
		"total_nodes":  len(clusterManager.ListNodes()),
	}

	writeJSON(w, http.StatusOK, health)
}

// =============================================================================
// Global Cluster Manager
// =============================================================================

var clusterManager *ClusterManager
