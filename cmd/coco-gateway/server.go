package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coco-sandbox/coco/pkg/types"
)

type GatewayServer struct {
	masterAddr string
	client     *http.Client
	nodeAddrs  []string
}

func NewGatewayServer(masterAddr string, nodeAddrs []string) *GatewayServer {
	return &GatewayServer{
		masterAddr: masterAddr,
		nodeAddrs:  nodeAddrs,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *GatewayServer) masterURL(path string) string {
	return "http://" + g.masterAddr + path
}

func (g *GatewayServer) CreateSandbox(ctx context.Context, req *types.CreateSandboxRequest) (*types.CreateSandboxResponse, error) {
	return doPost[types.CreateSandboxResponse](ctx, g.client, g.masterURL("/internal/v1/sandboxes"), req)
}

func (g *GatewayServer) GetSandbox(ctx context.Context, id string) (*types.Sandbox, error) {
	resp, err := doGet[types.GetSandboxResponse](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+id))
	if err != nil {
		return nil, err
	}
	return resp.Sandbox, nil
}

func (g *GatewayServer) ListSandboxes(ctx context.Context) (*types.ListSandboxesResponse, error) {
	return doGet[types.ListSandboxesResponse](ctx, g.client, g.masterURL("/internal/v1/sandboxes"))
}

func (g *GatewayServer) DeleteSandbox(ctx context.Context, id string) error {
	return doDelete(ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+id))
}

func (g *GatewayServer) PauseSandbox(ctx context.Context, id string) error {
	_, err := doPost[map[string]any](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+id+"/pause"), nil)
	return err
}

func (g *GatewayServer) ResumeSandbox(ctx context.Context, id string) error {
	_, err := doPost[map[string]any](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+id+"/resume"), nil)
	return err
}

func (g *GatewayServer) ForkSandbox(ctx context.Context, parentID string, req *types.ForkSandboxRequest) (*types.ForkSandboxResponse, error) {
	return doPost[types.ForkSandboxResponse](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+parentID+"/fork"), req)
}

func (g *GatewayServer) ExecSandbox(ctx context.Context, req *types.ExecSandboxRequest) ([]byte, int32, error) {
	type execResp struct {
		Output   []byte `json:"output"`
		ExitCode int32  `json:"exit_code"`
	}
	resp, err := doPost[execResp](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+req.SandboxID+"/exec"), req)
	if err != nil {
		return nil, 1, err
	}
	return resp.Output, resp.ExitCode, nil
}

func (g *GatewayServer) Create(ctx context.Context, req *types.CreateSandboxRequest) (*types.Sandbox, error) {
	resp, err := g.CreateSandbox(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Sandbox, nil
}

func (g *GatewayServer) Get(ctx context.Context, id string) (*types.Sandbox, error) {
	return g.GetSandbox(ctx, id)
}

func (g *GatewayServer) List(ctx context.Context) ([]*types.Sandbox, error) {
	resp, err := g.ListSandboxes(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Sandboxes, nil
}

func (g *GatewayServer) Update(ctx context.Context, id string, sb *types.Sandbox) (*types.Sandbox, error) {
	return sb, nil
}

func (g *GatewayServer) Delete(ctx context.Context, id string) error {
	return g.DeleteSandbox(ctx, id)
}

func (g *GatewayServer) Pause(ctx context.Context, id string) error {
	return g.PauseSandbox(ctx, id)
}

func (g *GatewayServer) Resume(ctx context.Context, id string) error {
	return g.ResumeSandbox(ctx, id)
}

func (g *GatewayServer) Fork(ctx context.Context, parentID string, req *types.ForkRequest) (*types.Sandbox, error) {
	forkReq := &types.ForkSandboxRequest{
		ParentID: parentID,
		Name:     req.Name,
		Labels:   req.Labels,
	}
	resp, err := g.ForkSandbox(ctx, parentID, forkReq)
	if err != nil {
		return nil, err
	}
	return resp.Sandbox, nil
}

func (g *GatewayServer) Hibernate(ctx context.Context, id string) error {
	_, err := doPost[map[string]any](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+id+"/hibernate"), nil)
	return err
}

func (g *GatewayServer) ResumeHibernate(ctx context.Context, id string) error {
	_, err := doPost[map[string]any](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+id+"/resume-hibernate"), nil)
	return err
}

// Cluster operations
func (g *GatewayServer) GetClusterInfo(ctx context.Context) (*types.ClusterInfoResponse, error) {
	return doGet[types.ClusterInfoResponse](ctx, g.client, g.masterURL("/internal/v1/cluster"))
}

func (g *GatewayServer) ListNodes(ctx context.Context) (*types.ListNodesResponse, error) {
	return doGet[types.ListNodesResponse](ctx, g.client, g.masterURL("/internal/v1/nodes"))
}

func (g *GatewayServer) GetNode(ctx context.Context, id string) (*types.NodeInfo, error) {
	resp, err := doGet[types.GetNodeResponse](ctx, g.client, g.masterURL("/internal/v1/nodes/"+id))
	if err != nil {
		return nil, err
	}
	return &resp.Node, nil
}

func (g *GatewayServer) DrainNode(ctx context.Context, id string) error {
	return doDelete(ctx, g.client, g.masterURL("/internal/v1/nodes/"+id+"/drain"))
}

// Template operations
func (g *GatewayServer) CreateTemplate(ctx context.Context, req *types.CreateTemplateRequest) (*types.Template, error) {
	resp, err := doPost[types.CreateTemplateResponse](ctx, g.client, g.masterURL("/internal/v1/templates"), req)
	if err != nil {
		return nil, err
	}
	return resp.Template, nil
}

func (g *GatewayServer) GetTemplate(ctx context.Context, id string) (*types.Template, error) {
	resp, err := doGet[types.GetTemplateResponse](ctx, g.client, g.masterURL("/internal/v1/templates/"+id))
	if err != nil {
		return nil, err
	}
	return resp.Template, nil
}

func (g *GatewayServer) ListTemplates(ctx context.Context) (*types.ListTemplatesResponse, error) {
	return doGet[types.ListTemplatesResponse](ctx, g.client, g.masterURL("/internal/v1/templates"))
}

func (g *GatewayServer) DeleteTemplate(ctx context.Context, id string) error {
	return doDelete(ctx, g.client, g.masterURL("/internal/v1/templates/"+id))
}

func (g *GatewayServer) BuildTemplate(ctx context.Context, id string, req *types.BuildTemplateRequest) error {
	_, err := doPost[map[string]any](ctx, g.client, g.masterURL("/internal/v1/templates/"+id+"/build"), req)
	return err
}

// Checkpoint operations
func (g *GatewayServer) CreateCheckpoint(ctx context.Context, sandboxID string, req *types.CreateCheckpointRequest) (*types.Checkpoint, error) {
	resp, err := doPost[types.CreateCheckpointResponse](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+sandboxID+"/checkpoints"), req)
	if err != nil {
		return nil, err
	}
	return resp.Checkpoint, nil
}

func (g *GatewayServer) ListCheckpoints(ctx context.Context, sandboxID string) (*types.ListCheckpointsResponse, error) {
	return doGet[types.ListCheckpointsResponse](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+sandboxID+"/checkpoints"))
}

func (g *GatewayServer) GetCheckpoint(ctx context.Context, id string) (*types.Checkpoint, error) {
	resp, err := doGet[types.GetCheckpointResponse](ctx, g.client, g.masterURL("/internal/v1/checkpoints/"+id))
	if err != nil {
		return nil, err
	}
	return resp.Checkpoint, nil
}

func (g *GatewayServer) DeleteCheckpoint(ctx context.Context, id string) error {
	return doDelete(ctx, g.client, g.masterURL("/internal/v1/checkpoints/"+id))
}

func (g *GatewayServer) RestoreSandbox(ctx context.Context, sandboxID string, checkpointID string) error {
	req := map[string]string{"checkpoint_id": checkpointID}
	_, err := doPost[map[string]any](ctx, g.client, g.masterURL("/internal/v1/sandboxes/"+sandboxID+"/restore"), req)
	return err
}

func doPost[T any](ctx context.Context, client *http.Client, url string, body any) (*T, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("master request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("master returned %d", resp.StatusCode)
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func doGet[T any](ctx context.Context, client *http.Client, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("master request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("master returned %d", resp.StatusCode)
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func doDelete(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("master request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("master returned %d", resp.StatusCode)
	}
	return nil
}
