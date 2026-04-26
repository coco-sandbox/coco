package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coco-sandbox/coco/pkg/scheduler"
	"github.com/coco-sandbox/coco/pkg/types"
)

type Balancer struct {
	scheduler *scheduler.Scheduler
	nodes     map[string]*NodeClient
	mu        sync.RWMutex
	strategy  LoadBalanceStrategy
}

type LoadBalanceStrategy int

const (
	StrategyRoundRobin LoadBalanceStrategy = iota
	StrategyLeastConnections
	StrategyLeastResponseTime
	StrategyWeighted
)

type NodeClient struct {
	ID            string
	Addr          string
	conn          interface{}
	activeConns   int
	totalConns    int64
	responseTimes []time.Duration
	mu            sync.Mutex
}

func NewBalancer(sched *scheduler.Scheduler) *Balancer {
	return &Balancer{
		scheduler: sched,
		nodes:     make(map[string]*NodeClient),
		strategy:  StrategyLeastConnections,
	}
}

func (b *Balancer) AddNode(nodeID, addr string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.nodes[nodeID]; !ok {
		b.nodes[nodeID] = &NodeClient{
			ID:   nodeID,
			Addr: addr,
		}
	}
}

func (b *Balancer) RemoveNode(nodeID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.nodes, nodeID)
}

func (b *Balancer) GetNode(ctx context.Context, sandboxID string) (*NodeClient, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var candidates []*NodeClient
	for _, node := range b.nodes {
		if node.activeConns == 0 {
			candidates = append(candidates, node)
		}
	}

	if len(candidates) == 0 {
		candidates = make([]*NodeClient, 0, len(b.nodes))
		for _, node := range b.nodes {
			candidates = append(candidates, node)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	var selected *NodeClient
	switch b.strategy {
	case StrategyRoundRobin:
		selected = b.roundRobin(candidates)
	case StrategyLeastConnections:
		selected = b.leastConnections(candidates)
	case StrategyLeastResponseTime:
		selected = b.leastResponseTime(candidates)
	default:
		selected = b.leastConnections(candidates)
	}

	selected.mu.Lock()
	selected.activeConns++
	selected.totalConns++
	selected.mu.Unlock()

	return selected, nil
}

func (b *Balancer) ReleaseNode(nodeID string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if node, ok := b.nodes[nodeID]; ok {
		node.mu.Lock()
		if node.activeConns > 0 {
			node.activeConns--
		}
		node.mu.Unlock()
	}
}

func (b *Balancer) RecordResponseTime(nodeID string, duration time.Duration) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if node, ok := b.nodes[nodeID]; ok {
		node.mu.Lock()
		node.responseTimes = append(node.responseTimes, duration)
		if len(node.responseTimes) > 100 {
			node.responseTimes = node.responseTimes[1:]
		}
		node.mu.Unlock()
	}
}

func (b *Balancer) roundRobin(nodes []*NodeClient) *NodeClient {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[time.Now().UnixNano()%int64(len(nodes))]
}

func (b *Balancer) leastConnections(nodes []*NodeClient) *NodeClient {
	var selected *NodeClient
	for _, node := range nodes {
		if selected == nil || node.activeConns < selected.activeConns {
			selected = node
		}
	}
	return selected
}

func (b *Balancer) leastResponseTime(nodes []*NodeClient) *NodeClient {
	var selected *NodeClient
	var minAvg time.Duration

	for _, node := range nodes {
		node.mu.Lock()
		avg := b.averageResponseTime(node.responseTimes)
		node.mu.Unlock()

		if selected == nil || avg < minAvg {
			selected = node
			minAvg = avg
		}
	}
	return selected
}

func (b *Balancer) averageResponseTime(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range times {
		total += t
	}
	return total / time.Duration(len(times))
}

func (b *Balancer) SetStrategy(strategy LoadBalanceStrategy) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.strategy = strategy
}

func (b *Balancer) GetStats() map[string]*BalancerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := make(map[string]*BalancerStats)
	for id, node := range b.nodes {
		node.mu.Lock()
		avgResponse := b.averageResponseTime(node.responseTimes)
		node.mu.Unlock()

		stats[id] = &BalancerStats{
			NodeID:          id,
			ActiveConns:     node.activeConns,
			TotalConns:      node.totalConns,
			AvgResponseTime: avgResponse,
		}
	}
	return stats
}

type BalancerStats struct {
	NodeID          string
	ActiveConns     int
	TotalConns      int64
	AvgResponseTime time.Duration
}

type RouteRequest struct {
	SandboxID string
	Method    string
	Payload   []byte
}

func (b *Balancer) Route(ctx context.Context, req *RouteRequest) ([]byte, error) {
	node, err := b.GetNode(ctx, req.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to route request: %w", err)
	}
	defer b.ReleaseNode(node.ID)

	start := time.Now()
	result, err := b.forwardRequest(ctx, node, req)
	duration := time.Since(start)

	b.RecordResponseTime(node.ID, duration)

	return result, err
}

func (b *Balancer) forwardRequest(ctx context.Context, node *NodeClient, req *RouteRequest) ([]byte, error) {
	return []byte{}, nil
}

func (b *Balancer) GetNodeBySandbox(sandboxID string) (string, error) {
	return "", nil
}

var _ types.BalancerServiceServer = &Balancer{}
