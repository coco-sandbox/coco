// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// RaftRole represents the Raft role of a node
type RaftRole string

const (
	RaftRoleFollower  RaftRole = "follower"
	RaftRoleCandidate RaftRole = "candidate"
	RaftRoleLeader    RaftRole = "leader"
)

// RaftNode represents a node in the Raft consensus
type RaftNode struct {
	ID            string    `json:"id"`
	Term          int64     `json:"term"`
	Role          RaftRole  `json:"role"`
	VotedFor      string    `json:"voted_for"`
	CommitIndex   int64     `json:"commit_index"`
	LastApplied   int64     `json:"last_applied"`
	Version      string    `json:"version"`
}

// RaftLogEntry represents a single entry in the Raft log
type RaftLogEntry struct {
	Index     int64     `json:"index"`
	Term      int64     `json:"term"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// RaftConsensus manages Raft consensus for the cluster
type RaftConsensus struct {
	nodeID     string
	nodes      map[string]*RaftNode
	log        []RaftLogEntry
	mu         sync.RWMutex
	role       RaftRole
	currentTerm int64
	votedFor    string
	commitIndex int64
	lastApplied int64

	// Leader address (for cross-node communication)
	leaderAddr string

	// Heartbeat timing
	heartbeatInterval time.Duration
	electionTimeout   time.Duration
	lastHeartbeat     time.Time

	// Vote tracking
	votesGranted   int
	votesNeeded    int

	// Cluster manager reference
	clusterMgr *ClusterManager

	stopCh chan struct{}
}

// NewRaftConsensus creates a new Raft consensus manager
func NewRaftConsensus(nodeID string, clusterMgr *ClusterManager) *RaftConsensus {
	return &RaftConsensus{
		nodeID:     nodeID,
		nodes:      make(map[string]*RaftNode),
		log:        []RaftLogEntry{},
		role:       RaftRoleFollower,
		currentTerm: 0,
		heartbeatInterval: 150 * time.Millisecond,
		electionTimeout:   300 * time.Millisecond,
		lastHeartbeat:     time.Now(),
		votesNeeded:       2, // Simple majority for 3 nodes
		clusterMgr: clusterMgr,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the Raft consensus loop
func (rc *RaftConsensus) Start() {
	log.Printf("[raft] Starting Raft consensus for node %s", rc.nodeID)
	rc.resetElectionTimer()
	go rc.consensusLoop()
}

// Stop shuts down Raft consensus
func (rc *RaftConsensus) Stop() {
	close(rc.stopCh)
}

// =============================================================================
// Core Raft Logic
// =============================================================================

func (rc *RaftConsensus) consensusLoop() {
	for {
		select {
		case <-rc.stopCh:
			return
		default:
			rc.tick()
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (rc *RaftConsensus) tick() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	switch rc.role {
	case RaftRoleFollower, RaftRoleCandidate:
		if time.Since(rc.lastHeartbeat) > rc.electionTimeout {
			rc.startElection()
		}
	case RaftRoleLeader:
		if time.Since(rc.lastHeartbeat) > rc.heartbeatInterval {
			rc.sendHeartbeat()
		}
	}
}

func (rc *RaftConsensus) resetElectionTimer() {
	rc.lastHeartbeat = time.Now()
}

func (rc *RaftConsensus) startElection() {
	rc.role = RaftRoleCandidate
	rc.currentTerm++
	rc.votedFor = rc.nodeID // Vote for self
	rc.votesGranted = 1    // Count self-vote
	rc.resetElectionTimer()

	log.Printf("[raft] Starting election for term %d", rc.currentTerm)

	// In a real implementation, send RequestVote to all other nodes
	// For skeleton, simulate getting votes
	aliveNodes := rc.clusterMgr.GetAliveNodes()
	rc.votesNeeded = (len(aliveNodes)/2)+1

	// Check if we won immediately (single node)
	if len(aliveNodes) <= 1 {
		rc.becomeLeader()
	}
}

func (rc *RaftConsensus) becomeLeader() {
	if rc.role != RaftRoleCandidate {
		return
	}

	rc.role = RaftRoleLeader
	log.Printf("[raft] Node %s became leader for term %d", rc.nodeID, rc.currentTerm)

	// Notify cluster manager
	rc.clusterMgr.ElectLeader()
}

func (rc *RaftConsensus) sendHeartbeat() {
	rc.lastHeartbeat = time.Now()

	// In a real implementation, send AppendEntries to all followers
	// For skeleton, just log
	if rc.role == RaftRoleLeader {
		log.Printf("[raft] Leader %s sending heartbeat for term %d", rc.nodeID, rc.currentTerm)
	}
}

// =============================================================================
// Log Operations
// =============================================================================

// AppendEntry adds a new entry to the Raft log
func (rc *RaftConsensus) AppendEntry(data []byte) (int64, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.role != RaftRoleLeader {
		return 0, fmt.Errorf("not leader (role: %s)", rc.role)
	}

	entry := RaftLogEntry{
		Index:     rc.lastApplied + 1,
		Term:      rc.currentTerm,
		Data:      data,
		Timestamp: time.Now(),
	}

	rc.log = append(rc.log, entry)
	rc.lastApplied++

	log.Printf("[raft] Appended entry %d to log (term: %d)", entry.Index, entry.Term)
	return entry.Index, nil
}

// GetLog returns the Raft log
func (rc *RaftConsensus) GetLog() []RaftLogEntry {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.log
}

// GetEntry returns a specific log entry
func (rc *RaftConsensus) GetEntry(index int64) (*RaftLogEntry, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	for i := range rc.log {
		if rc.log[i].Index == index {
			return &rc.log[i], nil
		}
	}
	return nil, fmt.Errorf("log entry %d not found", index)
}

// =============================================================================
// State Queries
// =============================================================================

// IsLeader returns true if this node is the Raft leader
func (rc *RaftConsensus) IsLeader() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.role == RaftRoleLeader
}

// GetRaftRole returns the current Raft role
func (rc *RaftConsensus) GetRaftRole() RaftRole {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.role
}

// GetTerm returns the current term
func (rc *RaftConsensus) GetTerm() int64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.currentTerm
}

// GetLeaderAddr returns the leader's address
func (rc *RaftConsensus) GetLeaderAddr() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.leaderAddr
}

// =============================================================================
// API Handlers
// =============================================================================

// HandleRaftStatus handles /raft/status endpoint
func handleRaftStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	raftConsensusMu.RLock()
	status := map[string]any{
		"node_id":      raft.nodeID,
		"role":         raft.role,
		"term":         raft.currentTerm,
		"voted_for":    raft.votedFor,
		"commit_index": raft.commitIndex,
		"last_applied": raft.lastApplied,
		"log_len":      len(raft.log),
		"is_leader":    raft.role == RaftRoleLeader,
	}
	raftConsensusMu.RUnlock()

	writeJSON(w, http.StatusOK, status)
}

// HandleRaftLog handles /raft/log endpoint
func handleRaftLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	raftConsensusMu.RLock()
	logEntries := raft.GetLog()
	raftConsensusMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": logEntries,
		"count":  len(logEntries),
	})
}

// HandleRaftPropose handles /raft/propose endpoint (add entry to log)
func handleRaftPropose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	raftConsensusMu.RLock()
	isLeader := raft.IsLeader()
	raftConsensusMu.RUnlock()

	if !isLeader {
		writeError(w, http.StatusServiceUnavailable, "not leader")
		return
	}

	// Read data from request body
	// For simplicity, use a map structure
	var req struct {
		Data string `json:"data"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	raftConsensusMu.Lock()
	index, err := raft.AppendEntry([]byte(req.Data))
	raftConsensusMu.Unlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to append: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"index": index,
		"term":  raft.currentTerm,
	})
}

// =============================================================================
// Global Raft Instance
// =============================================================================

var raft *RaftConsensus
var raftConsensusMu sync.RWMutex
