// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coco-sandbox/coco/pkg/pool"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/types"
	"github.com/coco-sandbox/coco/pkg/visor"
)

type NodeServer struct {
	store    *store.BadgerStore
	vmPool   *pool.Pool
	visor    *visor.Pool
	nodeID   string
	hostAddr string
}

func NewNodeServer(nodeID, hostAddr string, st *store.BadgerStore, vmPool *pool.Pool, visorPool *visor.Pool) *NodeServer {
	return &NodeServer{
		store:    st,
		vmPool:   vmPool,
		visor:    visorPool,
		nodeID:   nodeID,
		hostAddr: hostAddr,
	}
}

func (s *NodeServer) BootSandbox(ctx context.Context, req *types.BootSandboxRequest) (*types.BootSandboxResponse, error) {
	vm, err := s.vmPool.Acquire(ctx, req.SandboxID, req.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire VM: %w", err)
	}

	sb := &types.Sandbox{
		ID:        req.SandboxID,
		Name:      req.SandboxID,
		Template:  req.Template,
		MemoryMB:  req.MemoryMB,
		VCPUs:     req.VCPUs,
		State:     types.SandboxStateRunning,
		VsockCID:  vm.VsockCID,
		PID:       int32(vm.PID),
		HostNode:  s.nodeID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Labels:    make(map[string]string),
	}

	if err := s.store.PutSandbox(sb); err != nil {
		s.vmPool.Release(req.SandboxID)
		return nil, fmt.Errorf("failed to store sandbox: %w", err)
	}

	return &types.BootSandboxResponse{Sandbox: sb}, nil
}

func (s *NodeServer) DestroyVM(ctx context.Context, req *types.DestroyVMRequest) error {
	sb, err := s.store.GetSandbox(req.SandboxID)
	if err != nil {
		return fmt.Errorf("sandbox not found: %w", err)
	}

	if err := s.visor.Destroy(ctx, uint32(sb.PID)); err != nil {
		log.Printf("Warning: failed to destroy VM PID %d: %v", sb.PID, err)
	}

	if _, err := s.vmPool.Release(req.SandboxID); err != nil {
		log.Printf("Warning: failed to release VM from pool: %v", err)
	}

	if err := s.store.DeleteSandbox(req.SandboxID); err != nil {
		return fmt.Errorf("failed to delete sandbox from store: %w", err)
	}

	return nil
}

func (s *NodeServer) PauseVM(ctx context.Context, req *types.PauseVMRequest) error {
	sb, err := s.store.GetSandbox(req.SandboxID)
	if err != nil {
		return fmt.Errorf("sandbox not found: %w", err)
	}

	if err := s.visor.Pause(ctx, uint32(sb.PID)); err != nil {
		return fmt.Errorf("failed to pause VM: %w", err)
	}

	sb.State = types.SandboxStatePaused
	sb.UpdatedAt = time.Now()
	if err := s.store.PutSandbox(sb); err != nil {
		return fmt.Errorf("failed to update sandbox state: %w", err)
	}

	return nil
}

func (s *NodeServer) ResumeVM(ctx context.Context, req *types.ResumeVMRequest) error {
	sb, err := s.store.GetSandbox(req.SandboxID)
	if err != nil {
		return fmt.Errorf("sandbox not found: %w", err)
	}

	if err := s.visor.Resume(ctx, uint32(sb.PID)); err != nil {
		return fmt.Errorf("failed to resume VM: %w", err)
	}

	sb.State = types.SandboxStateRunning
	sb.UpdatedAt = time.Now()
	if err := s.store.PutSandbox(sb); err != nil {
		return fmt.Errorf("failed to update sandbox state: %w", err)
	}

	return nil
}

func (s *NodeServer) ExecVM(ctx context.Context, req *types.ExecRequest) ([]byte, int32, error) {
	sb, err := s.store.GetSandbox(req.SandboxID)
	if err != nil {
		return nil, 1, fmt.Errorf("sandbox not found: %w", err)
	}

	timeout := 30 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	output, exitCode, err := s.visor.Exec(ctx, uint32(sb.PID), req.Cmd, req.Args, req.Env, req.WorkingDir, timeout)
	if err != nil {
		return nil, 1, fmt.Errorf("exec failed: %w", err)
	}

	return output, int32(exitCode), nil
}

func (s *NodeServer) GetNodeStatus(ctx context.Context) (*types.NodeStatus, error) {
	stats := s.vmPool.Stats()

	var memUsed uint64
	sandboxCount := 0

	sandboxes, err := s.store.ListSandboxes()
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}

	for _, sb := range sandboxes {
		if sb.State == types.SandboxStateRunning {
			sandboxCount++
			memUsed += uint64(sb.MemoryMB)
		}
	}

	return &types.NodeStatus{
		NodeID:           s.nodeID,
		ActiveSandboxes:  int32(sandboxCount),
		PoolFree:         int32(stats.Free),
		MemoryUsedMB:     memUsed,
		CPUPercent:       0,
		Healthy:          true,
		LastUpdate:       time.Now(),
	}, nil
}

func (s *NodeServer) RegisterNode(ctx context.Context, nodeInfo *types.NodeInfo) error {
	if nodeInfo.NodeID != s.nodeID {
		return fmt.Errorf("node ID mismatch: expected %s, got %s", s.nodeID, nodeInfo.NodeID)
	}
	return nil
}
