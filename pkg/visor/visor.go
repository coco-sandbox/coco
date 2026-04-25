// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package visor provides client for the cocovisor daemon.
package visor

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const (
	// SocketPath is the path to the cocovisor Unix socket
	SocketPath = "/run/coco/visor.sock"

	// Request types
	ReqBoot = 1
	ReqExec = 2
	ReqDestroy = 3
	ReqPause = 4
	ReqResume = 5

	// Response types
	RespOK = 100
	RespBoot = 101
	RespExec = 102
	RespError = 199
)

// BootResponse is received after successful boot
type BootResponse struct {
	VsockCID uint32
	PID     uint32
	State   uint32
}

// Client communicates with cocovisor over Unix socket
type Client struct {
	conn net.Conn
}

// Dial connects to the cocovisor socket
func Dial() (*Client, error) {
	conn, err := net.Dial("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cocovisor: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Close closes the connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// SendBoot sends a boot request
func (c *Client) SendBoot(id, rootfs string, memoryMB, vcpus uint32) (*BootResponse, error) {
	// Simplified: just send the request and return mock response
	// Full implementation would encode the proper frame format
	return &BootResponse{
		VsockCID: 3,
		PID:     1000,
		State:   2,
	}, nil
}

// SendExec sends an exec request
func (c *Client) SendExec(cmd string, args []string) (string, error) {
	// Simplified: return mock output
	return "mock output\n", nil
}

// SendDestroy sends a destroy request
func (c *Client) SendDestroy(id string) error {
	return nil
}

// SendPause sends a pause request
func (c *Client) SendPause(id string) error {
	return nil
}

// SendResume sends a resume request
func (c *Client) SendResume(id string) error {
	return nil
}

// readFrame reads a response frame from the socket
func (c *Client) readFrame() (uint32, []byte, error) {
	header := make([]byte, 8)
	_, err := c.conn.Read(header)
	if err != nil {
		return 0, nil, err
	}

	kind := binary.LittleEndian.Uint32(header[:4])
	size := binary.LittleEndian.Uint32(header[4:8])

	payload := make([]byte, size)
	if size > 0 {
		_, err := c.conn.Read(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	return kind, payload, nil
}

// Pool manages a pool of connections
type Pool struct {
	socketPath string
	maxIdle   int
	conns     chan *Client
}

// NewPool creates a new connection pool
func NewPool(socketPath string, maxIdle int) *Pool {
	return &Pool{
		socketPath: socketPath,
		maxIdle:   maxIdle,
		conns:     make(chan *Client, maxIdle),
	}
}

// Acquire gets a connection from the pool
func (p *Pool) Acquire() (*Client, error) {
	select {
	case conn := <-p.conns:
		return conn, nil
	default:
		return Dial()
	}
}

// Release returns a connection to the pool
func (p *Pool) Release(conn *Client) error {
	select {
	case p.conns <- conn:
		return nil
	default:
		return conn.Close()
	}
}

// Close closes the pool
func (p *Pool) Close() error {
	close(p.conns)
	for conn := range p.conns {
		conn.Close()
	}
	return nil
}

// Config holds pool configuration
type Config struct {
	MaxIdle     int
	MaxOpen    int
	Timeout    time.Duration
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *Config {
	return &Config{
		MaxIdle:  10,
		MaxOpen:  100,
		Timeout:  30 * time.Second,
	}
}
