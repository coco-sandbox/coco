// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

import "time"

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleAgent    Role = "agent"
	RoleReadonly Role = "readonly"
)

type APIKey struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	Roles     []Role    `json:"roles"`
	Expires   int64     `json:"expires,omitempty"`
}

type ClusterInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	State         string    `json:"state"`
	NumNodes      int       `json:"num_nodes"`
	NumSandboxes  int       `json:"num_sandboxes"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	CreatedAt     time.Time `json:"created_at"`
}

// Node is an alias for NodeInfo for backwards compatibility
type Node = NodeInfo