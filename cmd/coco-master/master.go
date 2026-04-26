// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	connect "connectrpc.com/connect"
	v1 "github.com/coco-sandbox/coco/pkg/api/v1"
	"github.com/coco-sandbox/coco/pkg/api/v1/v1connect"
	"github.com/coco-sandbox/coco/pkg/scheduler"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func nodeEntryToProto(n *scheduler.NodeEntry) *v1.Node {
	return &v1.Node{
		Id:              n.ID,
		Addr:            n.Addr,
		Healthy:         n.Available,
		ActiveSandboxes: int32(n.Sandboxes),
		MemoryUsedMb:    n.MemMB,
		LastSeen:        timestamppb.New(n.UpdatedAt),
	}
}

var _ v1connect.MasterServiceHandler = (*MasterServer)(nil)

func unimplemented() error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("not yet implemented"))
}

type MasterServer struct {
	sched         *scheduler.Scheduler
	fm            *FailoverManager
	sandboxToNode map[string]string
	sandboxStates map[string]v1.SandboxState
	mu            sync.RWMutex
	election      *Election
}

func NewMasterServer(sched *scheduler.Scheduler, fm *FailoverManager) *MasterServer {
	return &MasterServer{
		sched:         sched,
		fm:            fm,
		sandboxToNode: make(map[string]string),
		sandboxStates: make(map[string]v1.SandboxState),
	}
}

func (s *MasterServer) SetElection(e *Election) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.election = e
}

func (s *MasterServer) IsLeader() bool {
	s.mu.RLock()
	e := s.election
	s.mu.RUnlock()
	if e == nil {
		return true
	}
	return e.IsLeader()
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

func (s *MasterServer) ScheduleSandbox(ctx context.Context, req *connect.Request[v1.ScheduleSandboxRequest]) (*connect.Response[v1.ScheduleSandboxResponse], error) {
	if !s.IsLeader() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("not leader"))
	}
	node, err := s.sched.Schedule(scheduler.StrategyLeastLoaded)
	if err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}
	sandboxID := generateID()
	s.mu.Lock()
	s.sandboxToNode[sandboxID] = node.ID
	s.mu.Unlock()
	return connect.NewResponse(&v1.ScheduleSandboxResponse{
		SandboxId:      sandboxID,
		AssignedNodeId: node.ID,
		AssignedAddr:   node.Addr,
	}), nil
}

func (s *MasterServer) GetSchedule(ctx context.Context, req *connect.Request[v1.GetScheduleRequest]) (*connect.Response[v1.GetScheduleResponse], error) {
	s.mu.RLock()
	nodeID, ok := s.sandboxToNode[req.Msg.SandboxId]
	s.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("sandbox %s not scheduled", req.Msg.SandboxId))
	}
	return connect.NewResponse(&v1.GetScheduleResponse{
		SandboxId:      req.Msg.SandboxId,
		AssignedNodeId: nodeID,
	}), nil
}

func (s *MasterServer) GetClusterInfo(ctx context.Context, req *connect.Request[v1.GetClusterInfoRequest]) (*connect.Response[v1.GetClusterInfoResponse], error) {
	nodes := s.sched.GetNodes()
	s.mu.RLock()
	sandboxCount := int32(len(s.sandboxToNode))
	s.mu.RUnlock()
	return connect.NewResponse(&v1.GetClusterInfoResponse{
		Cluster: &v1.ClusterInfo{
			NodeCount:    int32(len(nodes)),
			SandboxCount: sandboxCount,
			UpdatedAt:    timestamppb.Now(),
		},
	}), nil
}

func (s *MasterServer) ListNodes(ctx context.Context, req *connect.Request[v1.ListNodesRequest]) (*connect.Response[v1.ListNodesResponse], error) {
	nodes := s.sched.GetNodes()
	out := make([]*v1.Node, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, nodeEntryToProto(n))
	}
	return connect.NewResponse(&v1.ListNodesResponse{Nodes: out}), nil
}

func (s *MasterServer) GetNode(ctx context.Context, req *connect.Request[v1.GetNodeRequest]) (*connect.Response[v1.GetNodeResponse], error) {
	n, err := s.sched.GetNode(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.GetNodeResponse{Node: nodeEntryToProto(n)}), nil
}

func (s *MasterServer) DrainNode(ctx context.Context, req *connect.Request[v1.DrainNodeRequest]) (*connect.Response[v1.DrainNodeResponse], error) {
	if err := s.sched.DrainNode(req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.DrainNodeResponse{}), nil
}

func (s *MasterServer) RegisterLeader(ctx context.Context, req *connect.Request[v1.RegisterLeaderRequest]) (*connect.Response[v1.RegisterLeaderResponse], error) {
	return nil, unimplemented()
}

func (s *MasterServer) RequestVote(ctx context.Context, req *connect.Request[v1.RequestVoteRequest]) (*connect.Response[v1.RequestVoteResponse], error) {
	return nil, unimplemented()
}

func (s *MasterServer) TrackSandbox(ctx context.Context, req *connect.Request[v1.TrackSandboxRequest]) (*connect.Response[v1.TrackSandboxResponse], error) {
	sb := req.Msg.GetSandbox()
	if sb == nil || sb.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sandbox.id is required"))
	}
	s.mu.Lock()
	if hn := sb.GetHostNode(); hn != "" {
		s.sandboxToNode[sb.GetId()] = hn
	}
	s.sandboxStates[sb.GetId()] = sb.GetState()
	s.mu.Unlock()
	return connect.NewResponse(&v1.TrackSandboxResponse{}), nil
}

func (s *MasterServer) UntrackSandbox(ctx context.Context, req *connect.Request[v1.UntrackSandboxRequest]) (*connect.Response[v1.UntrackSandboxResponse], error) {
	id := req.Msg.GetSandboxId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sandbox_id is required"))
	}
	s.mu.Lock()
	delete(s.sandboxToNode, id)
	delete(s.sandboxStates, id)
	s.mu.Unlock()
	return connect.NewResponse(&v1.UntrackSandboxResponse{}), nil
}

func (s *MasterServer) UpdateSandboxState(ctx context.Context, req *connect.Request[v1.UpdateSandboxStateRequest]) (*connect.Response[v1.UpdateSandboxStateResponse], error) {
	id := req.Msg.GetSandboxId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sandbox_id is required"))
	}
	s.mu.Lock()
	if _, tracked := s.sandboxStates[id]; !tracked {
		s.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound, errors.New("sandbox not tracked"))
	}
	s.sandboxStates[id] = req.Msg.GetState()
	if hn := req.Msg.GetHostNode(); hn != "" {
		s.sandboxToNode[id] = hn
	}
	s.mu.Unlock()
	return connect.NewResponse(&v1.UpdateSandboxStateResponse{}), nil
}

func (s *MasterServer) InitiateFailover(ctx context.Context, req *connect.Request[v1.InitiateFailoverRequest]) (*connect.Response[v1.InitiateFailoverResponse], error) {
	if !s.IsLeader() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("not leader"))
	}
	nodeID := req.Msg.GetFailedNodeId()
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("failed_node_id is required"))
	}

	// Verify node exists
	if _, err := s.sched.GetNode(nodeID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("node %s not found: %w", nodeID, err))
	}

	// Register node failure; FailoverManager tracks node and triggers restore loop
	s.fm.RegisterNodeFailure(nodeID)

	// Collect sandboxes on this node and register each with FailoverManager
	s.mu.RLock()
	var toMigrate []string
	for sandboxID, sandboxNode := range s.sandboxToNode {
		if sandboxNode == nodeID {
			s.fm.RegisterSandboxFailure(sandboxID, nodeID)
			toMigrate = append(toMigrate, sandboxID)
		}
	}
	s.mu.RUnlock()

	return connect.NewResponse(&v1.InitiateFailoverResponse{
		Success:            true,
		SandboxesToMigrate:  toMigrate,
		MigrationCount:     int32(len(toMigrate)),
	}), nil
}

func (s *MasterServer) GetFailoverStatus(ctx context.Context, req *connect.Request[v1.GetFailoverStatusRequest]) (*connect.Response[v1.GetFailoverStatusResponse], error) {
	nodeID := req.Msg.GetNodeId()
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("node_id is required"))
	}

	state, migrated, pending := s.fm.GetNodeFailoverStatus(nodeID)

	return connect.NewResponse(&v1.GetFailoverStatusResponse{
		NodeId:             nodeID,
		State:              state,
		SandboxesMigrated:  migrated,
		SandboxesPending:   pending,
	}), nil
}
