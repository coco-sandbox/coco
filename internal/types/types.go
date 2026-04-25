// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package types provides shared types used across the Coco API server.
package types

import (
	"time"
)

// SandboxState represents the state of a sandbox
type SandboxState int

const (
	SandboxStateUnknown SandboxState = iota
	SandboxStateCreating
	SandboxStateRunning
	SandboxStatePaused
	SandboxStateStopping
	SandboxStateStopped
	SandboxStateHibernated
	SandboxStateError
)

func (s SandboxState) String() string {
	switch s {
	case SandboxStateCreating:
		return "creating"
	case SandboxStateRunning:
		return "running"
	case SandboxStatePaused:
		return "paused"
	case SandboxStateStopping:
		return "stopping"
	case SandboxStateStopped:
		return "stopped"
	case SandboxStateHibernated:
		return "hibernated"
	case SandboxStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Sandbox represents a Coco sandbox instance
type Sandbox struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	State         SandboxState   `json:"state"`
	Template      string         `json:"template"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	HostNode      string         `json:"host_node"`
	Config        map[string]any `json:"config,omitempty"`
	VsockCID      uint32         `json:"vsock_cid,omitempty"`
	PID           int            `json:"pid,omitempty"`
	Rootfs        string         `json:"rootfs,omitempty"`
	MemoryMB      int            `json:"memory_mb,omitempty"`
	VCPUs         int            `json:"vcpus,omitempty"`
	ParentID      string         `json:"parent_id,omitempty"`
	HibernatePath string         `json:"hibernate_path,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ForkDepth     int            `json:"fork_depth,omitempty"`
}

// Checkpoint represents a sandbox checkpoint
type Checkpoint struct {
	ID             string    `json:"id"`
	SandboxID      string    `json:"sandbox_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	ParentID       string    `json:"parent_id,omitempty"`
	Path           string    `json:"path"`
	SizeBytes      int64     `json:"size_bytes"`
	MemoryDiffMB   float32   `json:"memory_diff_mb"`
	StateSizeKB    uint32    `json:"state_size_kb"`
	CreatedAt      time.Time `json:"created_at"`
	Compression    string    `json:"compression"`
	IsRoot         bool      `json:"is_root"`
	ChainDepth     int       `json:"chain_depth"`
}

// Replay represents a replay session
type Replay struct {
	ID        string    `json:"id"`
	SandboxID string    `json:"sandbox_id"`
	State     string    `json:"state"`
	Events    int       `json:"events,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	StopTime  time.Time `json:"stop_time,omitempty"`
	Path      string    `json:"path,omitempty"`
}

// ClusterNode represents a node in the Coco cluster
type ClusterNode struct {
	ID        string    `json:"id"`
	Addr      string    `json:"addr"`
	State     string    `json:"state"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastSeen  time.Time `json:"last_seen"`
	Priority  int       `json:"priority"`
	Version   string    `json:"version"`
}

// Role defines user roles for RBAC
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleAgent    Role = "agent"
	RoleReadonly Role = "readonly"
)

// APIKey represents an API key for authentication
type APIKey struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	Roles     []Role   `json:"roles"`
	Expires   int64     `json:"expires,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ExecRequest represents a command execution request
type ExecRequest struct {
	Cmd        string   `json:"cmd"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	WorkingDir string   `json:"working_dir"`
	TimeoutMs  int64    `json:"timeout_ms,omitempty"`
}

// ExecChunk represents a chunk of exec output
type ExecChunk struct {
	StreamType int    `json:"stream_type"`
	Data       string `json:"data"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

// ForkRequest represents a fork request
type ForkRequest struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

// CreateSandboxRequest represents a request to create a sandbox
type CreateSandboxRequest struct {
	Name     string            `json:"name"`
	Template string            `json:"template"`
	MemoryMB int               `json:"memory_mb"`
	VCPUs    int               `json:"vcpus"`
	Labels   map[string]string `json:"labels,omitempty"`
}
