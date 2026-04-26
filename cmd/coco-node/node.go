package main

import (
	"context"
	"fmt"
	"time"

	connect "connectrpc.com/connect"
	v1 "github.com/coco-sandbox/coco/pkg/api/v1"
	"github.com/coco-sandbox/coco/pkg/api/v1/v1connect"
	"github.com/coco-sandbox/coco/pkg/cluster"
	"github.com/coco-sandbox/coco/pkg/pool"
	"github.com/coco-sandbox/coco/pkg/store"
	"github.com/coco-sandbox/coco/pkg/types"
	"github.com/coco-sandbox/coco/pkg/visor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ v1connect.NodeServiceHandler = (*NodeServer)(nil)

type NodeServer struct {
	store     *store.BadgerStore
	vmPool    *pool.Pool
	visorPool *visor.Pool
	discovery *cluster.Discovery
	nodeID    string
	hostAddr  string
}

func NewNodeServer(nodeID, hostAddr string, st *store.BadgerStore, vmPool *pool.Pool, vp *visor.Pool) *NodeServer {
	return &NodeServer{
		store:     st,
		vmPool:    vmPool,
		visorPool: vp,
		nodeID:    nodeID,
		hostAddr:  hostAddr,
	}
}

func (s *NodeServer) SetDiscovery(d *cluster.Discovery) {
	s.discovery = d
}

func (s *NodeServer) BootSandbox(ctx context.Context, req *connect.Request[v1.BootSandboxRequest]) (*connect.Response[v1.BootSandboxResponse], error) {
	vm, err := s.vmPool.Acquire(ctx, req.Msg.SandboxId, req.Msg.TemplateId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("acquire vm: %w", err))
	}
	return connect.NewResponse(&v1.BootSandboxResponse{
		SandboxId: vm.ID,
		Pid:       int32(vm.PID),
		VsockCid:  vm.VsockCID,
	}), nil
}

func (s *NodeServer) DestroyVM(ctx context.Context, req *connect.Request[v1.DestroyVMRequest]) (*connect.Response[v1.DestroyVMResponse], error) {
	s.vmPool.Release(req.Msg.SandboxId)
	if err := s.store.DeleteSandbox(req.Msg.SandboxId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DestroyVMResponse{}), nil
}

func (s *NodeServer) PauseVM(ctx context.Context, req *connect.Request[v1.PauseVMRequest]) (*connect.Response[v1.PauseVMResponse], error) {
	sb, err := s.store.GetSandbox(req.Msg.SandboxId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	client, err := s.visorPool.Acquire()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer s.visorPool.Release(client)
	if err := client.Pause(sb.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sb.State = types.SandboxStatePaused
	sb.UpdatedAt = time.Now()
	_ = s.store.PutSandbox(sb)
	return connect.NewResponse(&v1.PauseVMResponse{}), nil
}

func (s *NodeServer) ResumeVM(ctx context.Context, req *connect.Request[v1.ResumeVMRequest]) (*connect.Response[v1.ResumeVMResponse], error) {
	sb, err := s.store.GetSandbox(req.Msg.SandboxId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	client, err := s.visorPool.Acquire()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer s.visorPool.Release(client)
	if err := client.Resume(sb.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sb.State = types.SandboxStateRunning
	sb.UpdatedAt = time.Now()
	_ = s.store.PutSandbox(sb)
	return connect.NewResponse(&v1.ResumeVMResponse{}), nil
}

func (s *NodeServer) ForkVM(ctx context.Context, req *connect.Request[v1.ForkVMRequest]) (*connect.Response[v1.ForkVMResponse], error) {
	client, err := s.visorPool.Acquire()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer s.visorPool.Release(client)
	resp, err := client.Fork(req.Msg.ParentSandboxId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ForkVMResponse{
		SandboxId: req.Msg.NewSandboxId,
		VsockCid:  resp.ChildVsockCID,
	}), nil
}

func (s *NodeServer) ExecVM(ctx context.Context, req *connect.Request[v1.ExecVMRequest]) (*connect.Response[v1.ExecVMResponse], error) {
	client, err := s.visorPool.Acquire()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer s.visorPool.Release(client)
	var stdout []byte
	var exitCode uint32
	if err := client.Exec(visor.ExecRequest{
		SandboxID:  req.Msg.SandboxId,
		Cmd:        req.Msg.Cmd,
		Args:       req.Msg.Args,
		Env:        req.Msg.Env,
		WorkingDir: req.Msg.WorkingDir,
	}, func(c visor.ExecChunk) error {
		if c.StreamType == 3 {
			exitCode = c.ExitCode
		} else {
			stdout = append(stdout, c.Data...)
		}
		return nil
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ExecVMResponse{
		Stdout:   stdout,
		ExitCode: int32(exitCode),
	}), nil
}

func (s *NodeServer) StreamingExecVM(ctx context.Context, req *connect.Request[v1.StreamingExecVMRequest], stream *connect.ServerStream[v1.ExecChunk]) error {
	client, err := s.visorPool.Acquire()
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer s.visorPool.Release(client)
	return client.Exec(visor.ExecRequest{
		SandboxID:  req.Msg.SandboxId,
		Cmd:        req.Msg.Cmd,
		Args:       req.Msg.Args,
		Env:        req.Msg.Env,
		WorkingDir: req.Msg.WorkingDir,
	}, func(c visor.ExecChunk) error {
		streamType := v1.StreamType(c.StreamType)
		return stream.Send(&v1.ExecChunk{
			StreamType: streamType,
			Data:       c.Data,
			ExitCode:   int32(c.ExitCode),
		})
	})
}

func (s *NodeServer) GetNodeStatus(ctx context.Context, req *connect.Request[v1.GetNodeStatusRequest]) (*connect.Response[v1.NodeStatus], error) {
	active, free := s.vmPool.Stats()
	return connect.NewResponse(&v1.NodeStatus{
		NodeId:          s.nodeID,
		ActiveSandboxes: int32(active),
		PoolFree:        int32(free),
		Healthy:         true,
		LastUpdate:      timestamppb.Now(),
	}), nil
}

func (s *NodeServer) RegisterNode(ctx context.Context, req *connect.Request[v1.RegisterNodeRequest]) (*connect.Response[v1.RegisterNodeResponse], error) {
	if s.discovery != nil {
		if err := s.discovery.RegisterNode(ctx, s.nodeID, s.hostAddr, nil); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&v1.RegisterNodeResponse{Success: true}), nil
}

func (s *NodeServer) Heartbeat(ctx context.Context, req *connect.Request[v1.HeartbeatRequest]) (*connect.Response[v1.HeartbeatResponse], error) {
	if s.discovery != nil {
		if err := s.discovery.RefreshLease(ctx, s.nodeID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&v1.HeartbeatResponse{Acknowledged: true}), nil
}

func (s *NodeServer) ReportMetrics(ctx context.Context, req *connect.Request[v1.ReportMetricsRequest]) (*connect.Response[v1.ReportMetricsResponse], error) {
	return connect.NewResponse(&v1.ReportMetricsResponse{}), nil
}

func (s *NodeServer) GetPoolStatus(ctx context.Context, req *connect.Request[v1.GetPoolStatusRequest]) (*connect.Response[v1.PoolStatus], error) {
	active, free := s.vmPool.Stats()
	return connect.NewResponse(&v1.PoolStatus{
		TotalSlots: int32(active + free),
		FreeSlots:  int32(free),
	}), nil
}

func (s *NodeServer) PrewarmPool(ctx context.Context, req *connect.Request[v1.PrewarmPoolRequest]) (*connect.Response[v1.PrewarmPoolResponse], error) {
	return connect.NewResponse(&v1.PrewarmPoolResponse{}), nil
}
