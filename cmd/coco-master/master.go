// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/coco-sandbox/coco/pkg/scheduler"
	"github.com/coco-sandbox/coco/pkg/types"
	"google.golang.org/grpc"
)

type MasterServer struct {
	sched           *scheduler.Scheduler
	sandboxToNode   map[string]string
	mu              sync.RWMutex
	nodeConnections map[string]*grpc.ClientConn
}

func NewMasterServer(sched *scheduler.Scheduler) *MasterServer {
	return &MasterServer{
		sched:           sched,
		sandboxToNode:   make(map[string]string),
		nodeConnections: make(map[string]*grpc.ClientConn),
	}
}

func (s *MasterServer) CreateSandbox(ctx context.Context, req *types.CreateSandboxRequest) (*types.CreateSandboxResponse, error) {
	node, err := s.sched.Schedule(scheduler.StrategyLeastLoaded)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule sandbox: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}

	bootReq := &types.BootSandboxRequest{
		SandboxID: generateID(),
		Template:  req.Template,
		MemoryMB:  req.MemoryMB,
		VCPUs:     req.VCPUs,
	}

	sb := &types.Sandbox{
		ID:       bootReq.SandboxID,
		Name:     req.Name,
		Template: req.Template,
		MemoryMB: req.MemoryMB,
		VCPUs:    req.VCPUs,
		State:    types.SandboxStateCreating,
		Labels:   req.Labels,
		HostNode: node.ID,
	}

	s.mu.Lock()
	s.sandboxToNode[sb.ID] = node.ID
	s.mu.Unlock()

	_ = conn

	return &types.CreateSandboxResponse{Sandbox: sb}, nil
}

func (s *MasterServer) GetSandbox(ctx context.Context, req *types.GetSandboxRequest) (*types.GetSandboxResponse, error) {
	s.mu.RLock()
	nodeID, ok := s.sandboxToNode[req.ID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("sandbox not found")
	}

	node, err := s.sched.GetNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}

	_ = conn

	return &types.GetSandboxResponse{
		Sandbox: &types.Sandbox{
			ID: req.ID,
		},
	}, nil
}

func (s *MasterServer) ListSandboxes(ctx context.Context) (*types.ListSandboxesResponse, error) {
	return &types.ListSandboxesResponse{
		Sandboxes: []*types.Sandbox{},
	}, nil
}

func (s *MasterServer) DeleteSandbox(ctx context.Context, req *types.DeleteSandboxRequest) error {
	s.mu.Lock()
	nodeID, ok := s.sandboxToNode[req.ID]
	delete(s.sandboxToNode, req.ID)
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("sandbox not found")
	}

	node, err := s.sched.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return fmt.Errorf("failed to connect to node: %w", err)
	}

	_ = conn

	return nil
}

func (s *MasterServer) PauseSandbox(ctx context.Context, req *types.PauseSandboxRequest) error {
	s.mu.RLock()
	nodeID, ok := s.sandboxToNode[req.ID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("sandbox not found")
	}

	node, err := s.sched.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return fmt.Errorf("failed to connect to node: %w", err)
	}

	_ = conn

	return nil
}

func (s *MasterServer) ResumeSandbox(ctx context.Context, req *types.ResumeSandboxRequest) error {
	s.mu.RLock()
	nodeID, ok := s.sandboxToNode[req.ID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("sandbox not found")
	}

	node, err := s.sched.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return fmt.Errorf("failed to connect to node: %w", err)
	}

	_ = conn

	return nil
}

func (s *MasterServer) ForkSandbox(ctx context.Context, req *types.ForkSandboxRequest) (*types.ForkSandboxResponse, error) {
	s.mu.RLock()
	parentNodeID, ok := s.sandboxToNode[req.ParentID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("parent sandbox not found")
	}

	node, err := s.sched.GetNode(parentNodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}

	forkID := generateID()

	s.mu.Lock()
	s.sandboxToNode[forkID] = parentNodeID
	s.mu.Unlock()

	_ = conn

	return &types.ForkSandboxResponse{
		Sandbox: &types.Sandbox{
			ID:       forkID,
			Name:     req.Name,
			ParentID: req.ParentID,
		},
	}, nil
}

func (s *MasterServer) ExecSandbox(ctx context.Context, req *types.ExecSandboxRequest) ([]byte, int32, error) {
	s.mu.RLock()
	nodeID, ok := s.sandboxToNode[req.SandboxID]
	s.mu.RUnlock()

	if !ok {
		return nil, 1, fmt.Errorf("sandbox not found")
	}

	node, err := s.sched.GetNode(nodeID)
	if err != nil {
		return nil, 1, fmt.Errorf("node not found: %w", err)
	}

	conn, err := s.getOrCreateNodeConn(ctx, node.ID, node.Addr)
	if err != nil {
		return nil, 1, fmt.Errorf("failed to connect to node: %w", err)
	}

	_ = conn

	return []byte{}, 0, nil
}

func (s *MasterServer) getOrCreateNodeConn(ctx context.Context, nodeID, nodeAddr string) (*grpc.ClientConn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn, ok := s.nodeConnections[nodeID]; ok {
		return conn, nil
	}

	conn, err := grpc.DialContext(ctx, nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	s.nodeConnections[nodeID] = conn
	return conn, nil
}

func generateID() string {
	return "sb_" + randomString(24)
}

func randomString(length int) string {
	const charset = "0123456789abcdef"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
