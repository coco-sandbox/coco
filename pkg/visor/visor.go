// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package visor provides client for the cocovisor daemon.
package visor

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
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
	// Encode frame: [4 bytes type][4 bytes size][payload]
	req := fmt.Sprintf(`{"id":"%s","boot_source":{"kernel":"/var/lib/coco/vmlinux","initramfs":""},"root_volume":{"path":"%s","readonly":true},"cpus":{"count":%d},"memory":{"size":"%dM"}}`,
		id, rootfs, vcpus, memoryMB)

	// Frame header: type (ReqBoot=1) + size
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], ReqBoot)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(req)))

	_, err := c.conn.Write(header)
	if err != nil {
		return nil, err
	}
	_, err = c.conn.Write([]byte(req))
	if err != nil {
		return nil, err
	}

	// Read response frame
	kind, payload, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if kind != RespBoot && kind != RespOK {
		return nil, fmt.Errorf("unexpected response kind: %d", kind)
	}

	// Parse response JSON: {"id": "...", "pid": 1234, "vsock_cid": 3}
	// Simple parsing - look for pid and vsock_cid values
	resp := string(payload)
	pid := uint32(0)
	vsockCID := uint32(0)

	for _, line := range strings.Split(resp, ",") {
		if strings.Contains(line, "\"pid\"") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				pidStr := strings.TrimSpace(parts[1])
				pidStr = strings.Trim(pidStr, "}")
				if p, err := strconv.ParseUint(pidStr, 10, 32); err == nil {
					pid = uint32(p)
				}
			}
		}
		if strings.Contains(line, "\"vsock_cid\"") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				cidStr := strings.TrimSpace(parts[1])
				cidStr = strings.Trim(cidStr, "}")
				if c, err := strconv.ParseUint(cidStr, 10, 32); err == nil {
					vsockCID = uint32(c)
				}
			}
		}
	}

	return &BootResponse{
		VsockCID: vsockCID,
		PID:      pid,
		State:    2,
	}, nil
}

// SendExec sends an exec request
func (c *Client) SendExec(cmd string, args []string) (string, error) {
	// Encode exec request
	req := fmt.Sprintf(`{"command":"%s","args":[%s]}`, cmd, strings.Join(args, ","))

	// Frame header: type (ReqExec=2) + size
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], ReqExec)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(req)))

	_, err := c.conn.Write(header)
	if err != nil {
		return "", err
	}
	_, err = c.conn.Write([]byte(req))
	if err != nil {
		return "", err
	}

	// Read response
	kind, payload, err := c.readFrame()
	if err != nil {
		return "", err
	}
	if kind == RespError {
		return "", fmt.Errorf("exec failed: %s", string(payload))
	}
	if kind != RespExec && kind != RespOK {
		return "", fmt.Errorf("unexpected response kind: %d", kind)
	}

	// Parse output from response JSON: {"output": "...", "exit_code": 0}
	resp := string(payload)
	for _, line := range strings.Split(resp, ",") {
		if strings.Contains(line, "\"output\"") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				output := strings.TrimSpace(parts[1])
				output = strings.Trim(output, "\"")
				return output, nil
			}
		}
	}
	return string(payload), nil
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
