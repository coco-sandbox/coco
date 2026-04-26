// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	SocketPath = "/run/coco/visor.sock"

	// Request types
	ReqBoot             = 1
	ReqExec             = 2
	ReqDestroy          = 3
	ReqPause            = 4
	ReqResume           = 5
	ReqGetState         = 6
	ReqFork             = 7
	ReqHibernate        = 8
	ReqResumeHibernated = 9

	// Response types
	RespOK        = 100
	RespBoot      = 101
	RespExec      = 102
	RespDestroy   = 103
	RespGetState  = 106
	RespFork      = 107
	RespHibernate = 108
	RespError     = 199
)

type BootRequest struct {
	ID        string
	Rootfs    string
	Kernel    string
	Initrd    string
	MemoryMB  uint32
	VCPUs     uint32
	VsockPort uint32
}

type BootResponse struct {
	VsockCID uint32
	PID      uint32
	State    uint32
}

type ExecRequest struct {
	SandboxID  string
	Cmd        string
	Args       []string
	Env        []string
	WorkingDir string
}

type ExecChunk struct {
	StreamType uint32 // 1=stdout, 2=stderr, 3=exit
	Data       []byte
	ExitCode   uint32
}

type GetStateResponse struct {
	State    uint32
	PID      uint32
	VsockCID uint32
}

type ForkResponse struct {
	ChildVsockCID uint32
	ChildPID      uint32
	DurationMs    uint32
}

type Client struct {
	conn   net.Conn
	mu     sync.Mutex
	closed bool
}

func Dial() (*Client, error) {
	conn, err := net.DialTimeout("unix", SocketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cocovisor: %w", err)
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return c.conn.Close()
}

// Boot launches a VM from a template
func (c *Client) Boot(req BootRequest) (*BootResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var payload []byte
	idLen := uint32(len(req.ID))
	rootfsLen := uint32(len(req.Rootfs))
	kernelLen := uint32(len(req.Kernel))
	initrdLen := uint32(len(req.Initrd))

	total := 32 + int(idLen) + int(rootfsLen) + int(kernelLen) + int(initrdLen)
	payload = make([]byte, total)

	binary.LittleEndian.PutUint32(payload[0:4], rootfsLen)
	binary.LittleEndian.PutUint32(payload[4:8], req.MemoryMB)
	binary.LittleEndian.PutUint32(payload[8:12], req.VCPUs)
	binary.LittleEndian.PutUint32(payload[12:16], kernelLen)
	binary.LittleEndian.PutUint32(payload[16:20], initrdLen)
	binary.LittleEndian.PutUint32(payload[20:24], idLen)
	binary.LittleEndian.PutUint32(payload[24:28], req.VsockPort)
	binary.LittleEndian.PutUint32(payload[28:32], 0)

	off := 32
	copy(payload[off:], req.ID[:idLen])
	off += int(idLen)
	copy(payload[off:], req.Rootfs[:rootfsLen])
	off += int(rootfsLen)
	copy(payload[off:], req.Kernel[:kernelLen])
	off += int(kernelLen)
	copy(payload[off:], req.Initrd[:initrdLen])

	if err := c.sendFrame(ReqBoot, payload); err != nil {
		return nil, err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if kind == RespError {
		return nil, fmt.Errorf("visor error: %s", string(data))
	}
	if kind != RespBoot {
		return nil, fmt.Errorf("unexpected response kind: %d", kind)
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("boot response too short: got %d bytes, need 12", len(data))
	}

	return &BootResponse{
		VsockCID: binary.LittleEndian.Uint32(data[0:4]),
		PID:      binary.LittleEndian.Uint32(data[4:8]),
		State:    binary.LittleEndian.Uint32(data[8:12]),
	}, nil
}

// Exec runs a command in the VM with optional streaming callback
func (c *Client) Exec(req ExecRequest, cb func(ExecChunk) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	argsStr := strings.Join(req.Args, "\x00")
	envStr := strings.Join(req.Env, "\x00")
	sandboxIDLen := uint32(len(req.SandboxID))
	cmdLen := uint32(len(req.Cmd))
	argsLen := uint32(len(argsStr))
	envLen := uint32(len(envStr))
	workdirLen := uint32(len(req.WorkingDir))

	payload := make([]byte, 20+int(sandboxIDLen)+int(cmdLen)+int(argsLen)+int(envLen)+int(workdirLen))
	binary.LittleEndian.PutUint32(payload[0:4], sandboxIDLen)
	binary.LittleEndian.PutUint32(payload[4:8], cmdLen)
	binary.LittleEndian.PutUint32(payload[8:12], argsLen)
	binary.LittleEndian.PutUint32(payload[12:16], envLen)
	binary.LittleEndian.PutUint32(payload[16:20], workdirLen)
	off := 20
	copy(payload[off:], req.SandboxID)
	off += int(sandboxIDLen)
	copy(payload[off:], req.Cmd)
	off += int(cmdLen)
	copy(payload[off:], argsStr)
	off += int(argsLen)
	copy(payload[off:], envStr)
	off += int(envLen)
	copy(payload[off:], req.WorkingDir)

	if err := c.sendFrame(ReqExec, payload); err != nil {
		return err
	}

	for {
		kind, data, err := c.readFrame()
		if err != nil {
			return err
		}
		if kind == RespError {
			return fmt.Errorf("exec error: %s", string(data))
		}
		if kind != RespExec {
			return fmt.Errorf("unexpected response kind: %d", kind)
		}

		if len(data) < 12 {
			return fmt.Errorf("exec chunk too short: %d bytes", len(data))
		}
		chunk := ExecChunk{
			StreamType: binary.LittleEndian.Uint32(data[0:4]),
		}
		dataLen := binary.LittleEndian.Uint32(data[4:8])
		chunk.ExitCode = binary.LittleEndian.Uint32(data[8:12])
		if int(dataLen) > 0 && len(data) >= 12+int(dataLen) {
			chunk.Data = data[12 : 12+dataLen]
		}
		if err := cb(chunk); err != nil {
			return err
		}
		if chunk.StreamType == 3 {
			break
		}
	}

	return nil
}

// GetState returns current VM state
func (c *Client) GetState(sandboxID string) (*GetStateResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqGetState, []byte(sandboxID)); err != nil {
		return nil, err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if kind == RespError {
		return nil, fmt.Errorf("visor error: %s", string(data))
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("getstate response too short: got %d bytes, need 12", len(data))
	}

	return &GetStateResponse{
		State:    binary.LittleEndian.Uint32(data[0:4]),
		PID:      binary.LittleEndian.Uint32(data[4:8]),
		VsockCID: binary.LittleEndian.Uint32(data[8:12]),
	}, nil
}

// Destroy stops and destroys a VM
func (c *Client) Destroy(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqDestroy, []byte(id)); err != nil {
		return err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return err
	}
	if kind == RespError {
		return fmt.Errorf("visor error: %s", string(data))
	}

	return nil
}

// Pause suspends a running VM
func (c *Client) Pause(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqPause, []byte(id)); err != nil {
		return err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return err
	}
	if kind == RespError {
		return fmt.Errorf("visor error: %s", string(data))
	}

	return nil
}

// Resume resumes a paused VM
func (c *Client) Resume(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqResume, []byte(id)); err != nil {
		return err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return err
	}
	if kind == RespError {
		return fmt.Errorf("visor error: %s", string(data))
	}

	return nil
}

// Fork creates a fork of a running VM
func (c *Client) Fork(id string) (*ForkResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqFork, []byte(id)); err != nil {
		return nil, err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if kind == RespError {
		return nil, fmt.Errorf("visor error: %s", string(data))
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("fork response too short: got %d bytes, need 12", len(data))
	}

	return &ForkResponse{
		ChildVsockCID: binary.LittleEndian.Uint32(data[0:4]),
		ChildPID:      binary.LittleEndian.Uint32(data[4:8]),
		DurationMs:    binary.LittleEndian.Uint32(data[8:12]),
	}, nil
}

// Hibernate saves VM state to disk
func (c *Client) Hibernate(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqHibernate, []byte(id)); err != nil {
		return err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return err
	}
	if kind == RespError {
		return fmt.Errorf("visor error: %s", string(data))
	}
	if len(data) < 12 {
		return fmt.Errorf("hibernate response too short: got %d bytes, need 12", len(data))
	}

	return nil
}

// ResumeHibernated restores a hibernated VM
func (c *Client) ResumeHibernated(id string) (*BootResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.sendFrame(ReqResumeHibernated, []byte(id)); err != nil {
		return nil, err
	}

	kind, data, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if kind == RespError {
		return nil, fmt.Errorf("visor error: %s", string(data))
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("resumehibernated response too short: got %d bytes, need 12", len(data))
	}

	return &BootResponse{
		VsockCID: binary.LittleEndian.Uint32(data[0:4]),
		PID:      binary.LittleEndian.Uint32(data[4:8]),
		State:    binary.LittleEndian.Uint32(data[8:12]),
	}, nil
}

func (c *Client) sendFrame(kind uint32, payload []byte) error {
	frame := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(frame[0:4], kind)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(payload)))
	copy(frame[8:], payload)
	_, err := c.conn.Write(frame)
	return err
}

func (c *Client) readFrame() (uint32, []byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return 0, nil, err
	}
	kind := binary.LittleEndian.Uint32(header[0:4])
	size := binary.LittleEndian.Uint32(header[4:8])
	if size > 0 {
		data := make([]byte, size)
		if _, err := io.ReadFull(c.conn, data); err != nil {
			return 0, nil, err
		}
		return kind, data, nil
	}
	return kind, nil, nil
}

// Pool manages a pool of connections
type Pool struct {
	socketPath string
	maxIdle    int
	mu         sync.Mutex
	conns      chan *Client
}

func NewPool(socketPath string, maxIdle int) *Pool {
	return &Pool{
		socketPath: socketPath,
		maxIdle:    maxIdle,
		conns:      make(chan *Client, maxIdle),
	}
}

func (p *Pool) Acquire() (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case conn := <-p.conns:
		return conn, nil
	default:
		return Dial()
	}
}

func (p *Pool) Release(conn *Client) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case p.conns <- conn:
		return nil
	default:
		return conn.Close()
	}
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	close(p.conns)
	for conn := range p.conns {
		conn.Close()
	}
	return nil
}
