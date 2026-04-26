package strategies

import (
	"coco/pkg/scheduler"
)

type LeastLoadedStrategy struct{}

func NewLeastLoadedStrategy() *LeastLoadedStrategy {
	return &LeastLoadedStrategy{}
}

func (s *LeastLoadedStrategy) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	var selected *scheduler.NodeEntry
	var minLoad int = -1

	for _, node := range nodes {
		if selected == nil || node.Sandboxes < minLoad {
			selected = node
			minLoad = node.Sandboxes
		}
	}

	return selected
}

func (s *LeastLoadedStrategy) Name() string {
	return "least-loaded"
}

type LeastLoadedByMemory struct{}

func NewLeastLoadedByMemory() *LeastLoadedByMemory {
	return &LeastLoadedByMemory{}
}

func (s *LeastLoadedByMemory) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	var selected *scheduler.NodeEntry
	var minMemUsed uint64 = ^uint64(0)

	for _, node := range nodes {
		if selected == nil || node.MemMB < minMemUsed {
			selected = node
			minMemUsed = node.MemMB
		}
	}

	return selected
}

func (s *LeastLoadedByMemory) Name() string {
	return "least-loaded-memory"
}

type LeastCostStrategy struct {
	cpuCostPerUnit float64
	memCostPerMB   float64
}

func NewLeastCostStrategy(cpuCostPerUnit, memCostPerMB float64) *LeastCostStrategy {
	return &LeastCostStrategy{
		cpuCostPerUnit: cpuCostPerUnit,
		memCostPerMB:   memCostPerMB,
	}
}

func (s *LeastCostStrategy) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	var selected *scheduler.NodeEntry
	var minCost float64 = -1

	for _, node := range nodes {
		cost := float64(node.CPUs)*s.cpuCostPerUnit + float64(node.MemMB)*s.memCostPerMB
		if selected == nil || cost < minCost {
			selected = node
			minCost = cost
		}
	}

	return selected
}

func (s *LeastCostStrategy) Name() string {
	return "least-cost"
}
