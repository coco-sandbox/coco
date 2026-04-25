// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package coco

import "time"

// SandboxState represents the state of a sandbox
type SandboxState string

const (
	SandboxStateCreating   SandboxState = "creating"
	SandboxStateRunning   SandboxState = "running"
	SandboxStatePaused    SandboxState = "paused"
	SandboxStateHibernated SandboxState = "hibernated"
	SandboxStateStopping SandboxState = "stopping"
	SandboxStateStopped   SandboxState = "stopped"
	SandboxStateError     SandboxState = "error"
)

// Sandbox represents a Coco sandbox
type Sandbox struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	State     SandboxState     `json:"state"`
	Template  string            `json:"template"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	HostNode  string            `json:"host_node,omitempty"`
	VsockCID  uint32           `json:"vsock_cid,omitempty"`
	PID       int              `json:"pid,omitempty"`
	Rootfs    string            `json:"rootfs,omitempty"`
	MemoryMB  int              `json:"memory_mb,omitempty"`
	VCPUs     int              `json:"vcpus,omitempty"`
	ParentID  string            `json:"parent_id,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	ForkDepth int              `json:"fork_depth,omitempty"`
}

// SandboxConfig is used when creating a new sandbox
type SandboxConfig struct {
	Name     string            `json:"name,omitempty"`
	Template string            `json:"template,omitempty"`
	MemoryMB int              `json:"memory_mb,omitempty"`
	VCPUs    int              `json:"vcpus,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// Checkpoint represents a sandbox checkpoint
type Checkpoint struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SandboxID string    `json:"sandbox_id"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path,omitempty"`
	SizeBytes int64     `json:"size_bytes,omitempty"`
}

// ExecRequest is used when executing a command in a sandbox
type ExecRequest struct {
	Cmd      string            `json:"cmd"`
	Args     []string         `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	WorkingDir string          `json:"working_dir,omitempty"`
	TimeoutMs  int64           `json:"timeout_ms,omitempty"`
}

// ExecChunk represents a chunk of exec output
type ExecChunk struct {
	Type    string `json:"type"` // "stdout", "stderr", "exit"
	Data    string `json:"data,omitempty"`
	ExitCode int   `json:"exit_code,omitempty"`
}

// ForkRequest is used when forking a sandbox
type ForkRequest struct {
	Name   string            `json:"name,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

// ForkResponse is the response from a fork operation
type ForkResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	State        SandboxState `json:"state"`
	ParentID     string `json:"parent_id,omitempty"`
	ForkDurationMs int `json:"fork_duration_ms,omitempty"`
}

// HibernateResponse is the response from a hibernate operation
type HibernateResponse struct {
	State              SandboxState `json:"state"`
	SnapshotID         string       `json:"snapshot_id,omitempty"`
	HibernationDurationMs int64     `json:"hibernation_duration_ms,omitempty"`
}

// CreateSandboxResponse is the response from creating a sandbox
type CreateSandboxResponse struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	State     SandboxState  `json:"state"`
	VsockCID  uint32        `json:"vsock_cid,omitempty"`
	Sandbox   *Sandbox      `json:"sandbox,omitempty"`
}

// ListSandboxesResponse is the response from listing sandboxes
type ListSandboxesResponse struct {
	Items      []Sandbox `json:"items"`
	TotalCount int       `json:"total_count"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
	HasMore    bool      `json:"has_more"`
}

// DestroyResponse is the response from destroying a sandbox
type DestroyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// HealthResponse is the response from the health endpoint
type HealthResponse struct {
	Healthy bool   `json:"healthy"`
	Version string `json:"version,omitempty"`
}

// ReadyResponse is the response from the ready endpoint
type ReadyResponse struct {
	Ready  bool            `json:"ready"`
	Checks map[string]bool `json:"checks"`
}
