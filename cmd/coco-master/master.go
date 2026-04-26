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
	"github.com/coco-sandbox/coco/pkg/types"
	"google.golang.org/grpc"
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
	sched           *scheduler.Scheduler
	sandboxToNode   map[string]string
	sandboxStates   map[string]v1.SandboxState
	mu              sync.RWMutex
	nodeConnections map[string]*grpc.ClientConn
	election        *Election
}

func NewMasterServer(sched *scheduler.Scheduler) *MasterServer {
	return &MasterServer{
		sched:           sched,
		sandboxToNode:   make(map[string]string),
		sandboxStates:   make(map[string]v1.SandboxState),
		nodeConnections: make(map[string]*grpc.ClientConn),
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
	return nil, unimplemented()
}

func (s *MasterServer) GetFailoverStatus(ctx context.Context, req *connect.Request[v1.GetFailoverStatusRequest]) (*connect.Response[v1.GetFailoverStatusResponse], error) {
	return nil, unimplemented()
}
