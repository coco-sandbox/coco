package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type LocalScheduler struct {
	mu           sync.RWMutex
	queue        []string
	policy       SchedulingPolicy
	maxQueueSize int
}

type SchedulingPolicy int

const (
	PolicyFIFO SchedulingPolicy = iota
	PolicyPriority
	PolicyMemoryAware
)

func NewLocalScheduler(policy SchedulingPolicy, maxQueueSize int) *LocalScheduler {
	if maxQueueSize == 0 {
		maxQueueSize = 1000
	}

	return &LocalScheduler{
		policy:       policy,
		maxQueueSize: maxQueueSize,
		queue:        make([]string, 0),
	}
}

func (s *LocalScheduler) Enqueue(ctx context.Context, sandboxID string, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) >= s.maxQueueSize {
		return fmt.Errorf("scheduler queue is full")
	}

	s.queue = append(s.queue, sandboxID)

	if s.policy == PolicyPriority {
		s.sortByPriority(priority)
	}

	return nil
}

func (s *LocalScheduler) Dequeue() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return "", fmt.Errorf("queue is empty")
	}

	sandboxID := s.queue[0]
	s.queue = s.queue[1:]
	return sandboxID, nil
}

func (s *LocalScheduler) Peek() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.queue) == 0 {
		return "", fmt.Errorf("queue is empty")
	}

	return s.queue[0], nil
}

func (s *LocalScheduler) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.queue)
}

func (s *LocalScheduler) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = make([]string, 0)
}

func (s *LocalScheduler) SetPolicy(policy SchedulingPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
}

func (s *LocalScheduler) sortByPriority(priority int) {
}

type SandboxRequest struct {
	ID        string
	Priority  int
	CreatedAt time.Time
	MemoryMB uint64
	VCPUs    int
	Template string
}
