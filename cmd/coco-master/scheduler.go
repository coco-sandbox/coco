package main

import (
	"context"
	"fmt"
	"time"

	"coco/pkg/scheduler"
)

type MasterScheduler struct {
	*scheduler.Scheduler
	etcdEndpoints []string
	nodeTTL       time.Duration
}

func NewMasterScheduler(etcdEndpoints []string) *MasterScheduler {
	return &MasterScheduler{
		Scheduler:     scheduler.NewScheduler(),
		etcdEndpoints: etcdEndpoints,
		nodeTTL:       30 * time.Second,
	}
}

func (s *MasterScheduler) ScheduleSandbox(ctx context.Context, req *ScheduleRequest) (*scheduler.NodeEntry, error) {
	if req.MemoryMB == 0 {
		req.MemoryMB = 512
	}
	if req.VCPUs == 0 {
		req.VCPUs = 1
	}

	nodes := s.GetNodes()
	var candidates []*scheduler.NodeEntry
	for _, node := range nodes {
		if s.isNodeHealthy(node) && s.hasCapacity(node, req.MemoryMB, req.VCPUs) {
			candidates = append(candidates, node)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable nodes available")
	}

	node := s.selectNode(candidates, req.Strategy)
	if node == nil {
		return nil, fmt.Errorf("failed to select node")
	}

	return node, nil
}

func (s *MasterScheduler) isNodeHealthy(node *scheduler.NodeEntry) bool {
	if !node.Available {
		return false
	}
	return time.Since(node.UpdatedAt) < s.nodeTTL
}

func (s *MasterScheduler) hasCapacity(node *scheduler.NodeEntry, memMB uint64, vCPUs int) bool {
	availableMem := node.MemMB
	if availableMem < memMB {
		return false
	}
	return true
}

func (s *MasterScheduler) selectNode(nodes []*scheduler.NodeEntry, strategy scheduler.Strategy) *scheduler.NodeEntry {
	switch strategy {
	case scheduler.StrategyLeastLoaded:
		return s.leastLoadedNode(nodes)
	case scheduler.StrategyBinpack:
		return s.binpackNode(nodes)
	case scheduler.StrategyRandom:
		return s.randomNode(nodes)
	default:
		return s.leastLoadedNode(nodes)
	}
}

func (s *MasterScheduler) leastLoadedNode(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	var selected *scheduler.NodeEntry
	for _, node := range nodes {
		if selected == nil || node.Sandboxes < selected.Sandboxes {
			selected = node
		}
	}
	return selected
}

func (s *MasterScheduler) binpackNode(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	var selected *scheduler.NodeEntry
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

func (s *MasterScheduler) randomNode(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[time.Now().UnixNano()%int64(len(nodes))]
}

type ScheduleRequest struct {
	MemoryMB uint64
	VCPUs    int
	Labels   map[string]string
	Strategy scheduler.Strategy
}
