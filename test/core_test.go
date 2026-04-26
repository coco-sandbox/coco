// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"testing"
)

type SandboxState int

const (
	SandboxStateCreating SandboxState = iota
	SandboxStateRunning
	SandboxStatePaused
	SandboxStateHibernated
	SandboxStateStopping
	SandboxStateStopped
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
	case SandboxStateHibernated:
		return "hibernated"
	case SandboxStateStopping:
		return "stopping"
	case SandboxStateStopped:
		return "stopped"
	case SandboxStateError:
		return "error"
	default:
		return "unknown"
	}
}

func TestSandboxStateString(t *testing.T) {
	tests := []struct {
		state    SandboxState
		expected string
	}{
		{SandboxStateCreating, "creating"},
		{SandboxStateRunning, "running"},
		{SandboxStatePaused, "paused"},
		{SandboxStateHibernated, "hibernated"},
		{SandboxStateStopping, "stopping"},
		{SandboxStateStopped, "stopped"},
		{SandboxStateError, "error"},
		{SandboxState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("SandboxState.String() = %v, want %v", got, tt.expected)
		}
	}
}
