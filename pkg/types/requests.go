// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

import "time"

type CreateSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type GetSandboxRequest struct {
	ID string `json:"id"`
}

type GetSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type ListSandboxesResponse struct {
	Sandboxes []*Sandbox `json:"sandboxes"`
}

type DeleteSandboxRequest struct {
	ID string `json:"id"`
}

type PauseSandboxRequest struct {
	ID string `json:"id"`
}

type ResumeSandboxRequest struct {
	ID string `json:"id"`
}

type ForkSandboxRequest struct {
	ParentID string            `json:"parent_id"`
	Name     string            `json:"name"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type ForkSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type ExecSandboxRequest struct {
	SandboxID  string   `json:"sandbox_id"`
	Cmd        string   `json:"cmd"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	WorkingDir string   `json:"working_dir"`
	TimeoutMs  int64    `json:"timeout_ms,omitempty"`
}

type BootSandboxRequest struct {
	SandboxID string `json:"sandbox_id"`
	Template  string `json:"template"`
	MemoryMB  int    `json:"memory_mb"`
	VCPUs     int    `json:"vcpus"`
}

type BootSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type DestroyVMRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type PauseVMRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type ResumeVMRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type NodeStatus struct {
	NodeID          string    `json:"node_id"`
	ActiveSandboxes int32     `json:"active_sandboxes"`
	PoolFree        int32     `json:"pool_free"`
	MemoryUsedMB    uint64    `json:"memory_used_mb"`
	CPUPercent      int32     `json:"cpu_percent"`
	Healthy         bool      `json:"healthy"`
	LastUpdate      time.Time `json:"last_update"`
}

// =============================================================================
// Cluster Service Types (spec 2.4)
// =============================================================================

type ClusterInfoResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	State         string    `json:"state"`
	NumNodes      int       `json:"num_nodes"`
	NumSandboxes  int       `json:"num_sandboxes"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	CreatedAt     time.Time `json:"created_at"`
}

type GetNodeResponse struct {
	Node NodeInfo `json:"node"`
}

type ListNodesResponse struct {
	Items []*NodeInfo `json:"items"`
	Total int         `json:"total"`
}

// =============================================================================
// Template Service Types (spec 2.3)
// =============================================================================

type CreateTemplateResponse struct {
	Template *Template `json:"template"`
}

type GetTemplateResponse struct {
	Template *Template `json:"template"`
}

type ListTemplatesResponse struct {
	Items []*Template `json:"items"`
	Total int         `json:"total"`
}

type BuildTemplateRequest struct {
	Source     string `json:"source"`
	Dockerfile string `json:"dockerfile"`
}

type BuildTemplateResponse struct {
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
}

// =============================================================================
// Checkpoint Service Types (spec 2.1)
// =============================================================================

type CreateCheckpointRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Compression string `json:"compression,omitempty"`
}

type CreateCheckpointResponse struct {
	Checkpoint *Checkpoint `json:"checkpoint"`
}

type GetCheckpointResponse struct {
	Checkpoint *Checkpoint `json:"checkpoint"`
}

type ListCheckpointsResponse struct {
	Items []*Checkpoint `json:"items"`
	Total int            `json:"total"`
}

type RestoreCheckpointRequest struct {
	CheckpointID string `json:"checkpoint_id"`
}