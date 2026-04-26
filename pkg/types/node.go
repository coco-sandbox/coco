// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

import "time"

type NodeState string

const (
	NodeStateUnknown   NodeState = "unknown"
	NodeStateRunning   NodeState = "running"
	NodeStateUnhealthy NodeState = "unhealthy"
	NodeStateStopped   NodeState = "stopped"
)

type NodeInfo struct {
	ID        string    `json:"id"`
	Addr      string    `json:"addr"`
	State     NodeState `json:"state"`
	MemMB     uint64    `json:"mem_mb"`
	CPUs      int       `json:"cpus"`
	Sandboxes int       `json:"sandboxes"`
	Available bool      `json:"available"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastSeen  time.Time `json:"last_seen"`
}

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
