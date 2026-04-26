package strategies

import (

	"github.com/coco-sandbox/coco/pkg/scheduler"
)

type BinpackStrategy struct{}

func NewBinpackStrategy() *BinpackStrategy {
	return &BinpackStrategy{}
}

func (s *BinpackStrategy) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	var selected *scheduler.NodeEntry
	var maxLoad float64

	for _, node := range nodes {
		load := float64(node.Sandboxes)
		if selected == nil || load > maxLoad {
			selected = node
			maxLoad = load
		}
	}

	return selected
}

func (s *BinpackStrategy) Name() string {
	return "binpack"
}

type BinpackByMemory struct{}

func NewBinpackByMemory() *BinpackByMemory {
	return &BinpackByMemory{}
}

func (s *BinpackByMemory) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	var selected *scheduler.NodeEntry
	var maxMemUsed float64

	for _, node := range nodes {
		memUsed := float64(node.MemMB)
		if selected == nil || memUsed > maxMemUsed {
			selected = node
			maxMemUsed = memUsed
		}
	}

	return selected
}

func (s *BinpackByMemory) Name() string {
	return "binpack-memory"
}
