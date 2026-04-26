// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package scheduler

import (
	"testing"
	"time"
)

func TestScheduler_UpdateLoadFromReport(t *testing.T) {
	sched := NewScheduler()

	node := &NodeEntry{
		ID:        "node-1",
		Addr:      "localhost:9090",
		Sandboxes: 5,
		MemMB:     1024,
		CPUs:      4,
		Available: true,
		UpdatedAt: time.Now(),
	}

	if err := sched.RegisterNode(node); err != nil {
		t.Fatalf("RegisterNode: %v", err)
	}

	report := &LoadReport{
		NodeID:    "node-1",
		Sandboxes: 10,
		MemUsedMB: 2048,
		CPUs:      4,
		Timestamp: time.Now(),
	}

	sched.UpdateLoadFromReport(report)

	updated, err := sched.GetNode("node-1")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}

	if updated.Sandboxes != 10 {
		t.Errorf("Sandboxes = %d, want 10", updated.Sandboxes)
	}
	if updated.MemMB != 2048 {
		t.Errorf("MemMB = %d, want 2048", updated.MemMB)
	}
}

func TestScheduler_Schedule_StaleNode(t *testing.T) {
	sched := NewScheduler()

	// Register node normally first
	node := &NodeEntry{
		ID:        "node-stale",
		Addr:      "localhost:9090",
		Sandboxes: 0,
		Available: true,
		UpdatedAt: time.Now(),
	}
	sched.RegisterNode(node)

	// Force UpdatedAt to 31 seconds ago (past the 30s stale threshold)
	sched.UpdateLoadFromReport(&LoadReport{
		NodeID:    "node-stale",
		Sandboxes: 0,
		MemUsedMB: 0,
		CPUs:      0,
		Timestamp: time.Now().Add(-31 * time.Second),
	})

	_, err := sched.Schedule(StrategyLeastLoaded)
	if err == nil {
		t.Error("Schedule: expected error for stale node, got nil")
	}
}

func TestLoadReport(t *testing.T) {
	report := &LoadReport{
		NodeID:    "node-1",
		Sandboxes: 3,
		MemUsedMB: 512,
		CPUs:      2,
		Timestamp: time.Now(),
	}

	if report.NodeID != "node-1" {
		t.Errorf("NodeID = %s, want node-1", report.NodeID)
	}
	if report.Sandboxes != 3 {
		t.Errorf("Sandboxes = %d, want 3", report.Sandboxes)
	}
}
