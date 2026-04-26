// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

import "time"

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

type Sandbox struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	State         SandboxState      `json:"state"`
	Template      string            `json:"template"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	HostNode      string            `json:"host_node"`
	Config        map[string]any    `json:"config,omitempty"`
	VsockCID      uint32            `json:"vsock_cid,omitempty"`
	PID           int               `json:"pid,omitempty"`
	Rootfs        string            `json:"rootfs,omitempty"`
	MemoryMB      int               `json:"memory_mb,omitempty"`
	VCPUs         int               `json:"vcpus,omitempty"`
	ParentID      string            `json:"parent_id,omitempty"`
	HibernatePath string            `json:"hibernate_path,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ForkDepth     int               `json:"fork_depth,omitempty"`
}

type CreateSandboxRequest struct {
	Name     string            `json:"name"`
	Template string            `json:"template"`
	MemoryMB int               `json:"memory_mb"`
	VCPUs    int               `json:"vcpus"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type ForkRequest struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

type ExecRequest struct {
	Cmd        string   `json:"cmd"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	WorkingDir string   `json:"working_dir"`
	TimeoutMs  int64    `json:"timeout_ms,omitempty"`
}

type ExecChunk struct {
	StreamType int    `json:"stream_type"`
	Data       string `json:"data"`
	ExitCode   int    `json:"exit_code,omitempty"`
}
