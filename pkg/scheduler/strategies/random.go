package strategies

import (
	"math/rand"

	"coco/pkg/scheduler"
)

type RandomStrategy struct{}

func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{}
}

func (s *RandomStrategy) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	return nodes[rand.Intn(len(nodes))]
}

func (s *RandomStrategy) Name() string {
	return "random"
}

type RoundRobinStrategy struct {
	current int
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		current: -1,
	}
}

func (s *RoundRobinStrategy) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	s.current = (s.current + 1) % len(nodes)
	return nodes[s.current]
}

func (s *RoundRobinStrategy) Name() string {
	return "round-robin"
}

type FirstFitStrategy struct{}

func NewFirstFitStrategy() *FirstFitStrategy {
	return &FirstFitStrategy{}
}

func (s *FirstFitStrategy) Select(nodes []*scheduler.NodeEntry) *scheduler.NodeEntry {
	if len(nodes) == 0 {
		return nil
	}

	for _, node := range nodes {
		if node.Available {
			return node
		}
	}

	return nil
}

func (s *FirstFitStrategy) Name() string {
	return "first-fit"
}
