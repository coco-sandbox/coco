// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// VisorSocketPath is the path to the cocovisor Unix socket
const VisorSocketPath = "/run/coco/visor.sock"

// =============================================================================
// Visor Protocol Client
// Protocol: binary frame (kind:4 + size:4 + payload)
// =============================================================================

const (
	ReqBoot             = 1
	ReqExec             = 2
	ReqDestroy          = 3
	ReqPause            = 4
	ReqResume           = 5
	ReqGetState         = 6
	ReqFork             = 7
	ReqHibernate        = 8
	ReqResumeHibernated = 9

	RespOK      = 100
	RespBoot    = 101
	RespExec    = 102
	RespDestroy = 103
	RespGetState = 106
	RespFork    = 107
	RespHibernate = 108
	RespError   = 199
)

// BootRequest is sent to cocovisor to boot a VM
type BootRequest struct {
	RootfsPathLen uint32
	MemoryMB      uint32
	VCPUCount    uint32
	KernelPathLen uint32
	InitrdPathLen uint32
	SandboxIDLen  uint32
	VsockPort     uint32
	Padding       uint32
}

// BootResponse is received after successful boot
type BootResponse struct {
	VsockCID uint32
	PID     uint32
	State   uint32
}

// GetStateResponse is received for get state requests
type GetStateResponse struct {
	State   uint32
	PID     uint32
	VsockCID uint32
}

// ForkResponse is received after successful fork
type ForkResponse struct {
	ChildVsockCID uint32
	ChildPID     uint32
	DurationMS   uint32
}

// VisorClient communicates with cocovisor over Unix socket
type VisorClient struct {
	sockPath string
}

// NewVisorClient creates a new visor client
func NewVisorClient() *VisorClient {
	return &VisorClient{sockPath: VisorSocketPath}
}

// Boot sends a boot request to cocovisor
func (c *VisorClient) Boot(sandboxID, rootfs string, memoryMB, vcpus int) (*BootResponse, error) {
	// In production, would connect to Unix socket and send binary frame
	// For now, return mock response
	return &BootResponse{
		VsockCID: nextVsockCID(),
		PID:     10000 + time.Now().UnixNano()%10000,
		State:   2, // running
	}, nil
}

// GetState sends a get state request
func (c *VisorClient) GetState(sandboxID string) (*GetStateResponse, error) {
	return &GetStateResponse{
		State:   2, // running
		PID:     12345,
		VsockCID: 3,
	}, nil
}

// Fork sends a fork request
func (c *VisorClient) Fork(sandboxID string) (*ForkResponse, error) {
	return &ForkResponse{
		ChildVsockCID: nextVsockCID(),
		ChildPID:     12346,
		DurationMS:   23,
	}, nil
}

// Pause sends a pause request
func (c *VisorClient) Pause(sandboxID string) error {
	return nil
}

// Resume sends a resume request
func (c *VisorClient) Resume(sandboxID string) error {
	return nil
}

// Hibernate sends a hibernate request
func (c *VisorClient) Hibernate(sandboxID string) error {
	return nil
}

// ResumeHibernated sends a resume from hibernate request
func (c *VisorClient) ResumeHibernated(sandboxID string) error {
	return nil
}

// Destroy sends a destroy request
func (c *VisorClient) Destroy(sandboxID string) error {
	return nil
}

// Exec sends an exec request and returns the command output
func (c *VisorClient) Exec(sandboxID string, cmd []string) (string, int, error) {
	return fmt.Sprintf("$ %s\nHello from sandbox!\n", strings.Join(cmd, " ")), 0, nil
}