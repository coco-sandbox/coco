// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package scheduler

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Strategy int

const (
	StrategyLeastLoaded Strategy = iota
	StrategyBinpack
	StrategyRandom
)

type NodeEntry struct {
	ID        string
	Addr      string
	Sandboxes int
	MemMB     uint64
	CPUs      int
	Available bool
	UpdatedAt time.Time
	Labels    map[string]string
}

type ScheduleRequest struct {
	MemoryMB uint64
	VCPUs    int
}

type Scheduler struct {
	mu    sync.RWMutex
	nodes map[string]*NodeEntry
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		nodes: make(map[string]*NodeEntry),
	}
}

func (s *Scheduler) RegisterNode(entry *NodeEntry) error {
	if entry.ID == "" || entry.Addr == "" {
		return fmt.Errorf("node id and addr are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry.UpdatedAt = time.Now()
	s.nodes[entry.ID] = entry
	return nil
}

func (s *Scheduler) DeregisterNode(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nodes, id)
}

func (s *Scheduler) UpdateLoad(id string, sandboxes int, memUsedMB uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node, ok := s.nodes[id]; ok {
		node.Sandboxes = sandboxes
		node.MemMB = memUsedMB
		node.UpdatedAt = time.Now()
	}
}

func (s *Scheduler) Schedule(strategy Strategy) (*NodeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	available := make([]*NodeEntry, 0, len(s.nodes))
	for _, node := range s.nodes {
		if node.Available && time.Since(node.UpdatedAt) < 30*time.Second {
			available = append(available, node)
		}
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	switch strategy {
	case StrategyLeastLoaded:
		return s.leastLoaded(available), nil
	case StrategyBinpack:
		return s.binpack(available), nil
	case StrategyRandom:
		return available[rand.Intn(len(available))], nil
	default:
		return s.leastLoaded(available), nil
	}
}

func (s *Scheduler) leastLoaded(nodes []*NodeEntry) *NodeEntry {
	var selected *NodeEntry
	for _, node := range nodes {
		if selected == nil || node.Sandboxes < selected.Sandboxes {
			selected = node
		}
	}
	return selected
}

func (s *Scheduler) binpack(nodes []*NodeEntry) *NodeEntry {
	var selected *NodeEntry
	for _, node := range nodes {
		if selected == nil {
			selected = node
			continue
		}
		if node.Sandboxes > selected.Sandboxes {
			selected = node
		}
	}
	return selected
}

func (s *Scheduler) GetNodes() []*NodeEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*NodeEntry, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func (s *Scheduler) GetNode(id string) (*NodeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, ok := s.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found")
	}
	return node, nil
}
