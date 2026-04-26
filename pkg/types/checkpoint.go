// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

import "time"

type Checkpoint struct {
	ID           string    `json:"id"`
	SandboxID    string    `json:"sandbox_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ParentID     string    `json:"parent_id,omitempty"`
	Path         string    `json:"path"`
	SizeBytes    int64     `json:"size_bytes"`
	MemoryDiffMB float32   `json:"memory_diff_mb"`
	StateSizeKB  uint32    `json:"state_size_kb"`
	CreatedAt    time.Time `json:"created_at"`
	Compression  string    `json:"compression"`
	IsRoot       bool      `json:"is_root"`
	ChainDepth   int       `json:"chain_depth"`
}

type Replay struct {
	ID        string    `json:"id"`
	SandboxID string    `json:"sandbox_id"`
	State     string    `json:"state"`
	Events    int       `json:"events,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	StopTime  time.Time `json:"stop_time,omitempty"`
	Path      string    `json:"path,omitempty"`
}

type ReplayEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"`
}
