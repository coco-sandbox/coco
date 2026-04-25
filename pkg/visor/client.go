// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import (
    "encoding/binary"
    "fmt"
    "net"
    "sync"
    "time"
)

const (
    SocketPath = "/run/coco/visor.sock"

    // Request types
    ReqBoot              = 1
    ReqExec              = 2
    ReqDestroy           = 3
    ReqPause             = 4
    ReqResume            = 5
    ReqGetState          = 6
    ReqFork              = 7
    ReqHibernate         = 8
    ReqResumeHibernated  = 9

    // Response types
    RespOK               = 100
    RespBoot             = 101
    RespExec             = 102
    RespDestroy          = 103
    RespGetState         = 106
    RespFork             = 107
    RespHibernate        = 108
    RespError            = 199
)

type BootRequest struct {
    ID         string
    Rootfs     string
    Kernel     string
    Initrd     string
    MemoryMB   uint32
    VCPUs      uint32
    VsockPort  uint32
}

type BootResponse struct {
    VsockCID uint32
    PID      uint32
    State    uint32
}

type ExecRequest struct {
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

    // BootRequest: id_len(4) + mem(4) + vcpus(4) + rootfs_len(4) + kernel_len(4) + initrd_len(4) + vsock_port(4) + padding(4)
    payload = make([]byte, 36+int(idLen)+int(rootfsLen)+int(kernelLen)+int(initrdLen))
    binary.LittleEndian.PutUint32(payload[0:4], idLen)
    binary.LittleEndian.PutUint32(payload[4:8], req.MemoryMB)
    binary.LittleEndian.PutUint32(payload[8:12], req.VCPUs)
    binary.LittleEndian.PutUint32(payload[12:16], rootfsLen)
    binary.LittleEndian.PutUint32(payload[16:20], kernelLen)
    binary.LittleEndian.PutUint32(payload[20:24], initrdLen)
    binary.LittleEndian.PutUint32(payload[24:28], req.VsockPort)
    binary.LittleEndian.PutUint32(payload[28:32], 0) // padding

    off := 36
    copy(payload[off:off+int(idLen)], req.ID)
    off += int(idLen)
    copy(payload[off:off+int(rootfsLen)], req.Rootfs)
    off += int(rootfsLen)
    copy(payload[off:off+int(kernelLen)], req.Kernel)
    off += int(kernelLen)
    copy(payload[off:off+int(initrdLen)], req.Initrd)

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

    cmdLen := uint32(len(req.Cmd))
    argsStr := joinStrings(req.Args, "\x00")
    envStr := joinStrings(req.Env, "\x00")
    envLen := uint32(len(envStr))
    workdirLen := uint32(len(req.WorkingDir))

    var payload []byte
    totalLen := 16 + int(cmdLen) + len(argsStr) + int(envLen) + int(workdirLen)
    payload = make([]byte, totalLen)

    binary.LittleEndian.PutUint32(payload[0:4], cmdLen)
    binary.LittleEndian.PutUint32(payload[4:8], uint32(len(argsStr)))
    binary.LittleEndian.PutUint32(payload[8:12], envLen)
    binary.LittleEndian.PutUint32(payload[12:16], workdirLen)

    off := 16
    copy(payload[off:off+int(cmdLen)], req.Cmd)
    off += int(cmdLen)
    copy(payload[off:off+len(argsStr)], argsStr)
    off += len(argsStr)
    copy(payload[off:off+int(envLen)], envStr)
    off += int(envLen)
    copy(payload[off:off+int(workdirLen)], req.WorkingDir)

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

        chunk := ExecChunk{
            StreamType: binary.LittleEndian.Uint32(data[0:4]),
            Data:       data[8:],
        }

        if chunk.StreamType == 3 { // exit
            if len(data) >= 12 {
                chunk.ExitCode = binary.LittleEndian.Uint32(data[4:8])
            }
            break
        }

        if cb != nil {
            if err := cb(chunk); err != nil {
                return err
            }
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
    if _, err := c.conn.Read(header); err != nil {
        return 0, nil, err
    }
    kind := binary.LittleEndian.Uint32(header[0:4])
    size := binary.LittleEndian.Uint32(header[4:8])
    if size > 0 {
        data := make([]byte, size)
        if _, err := c.conn.Read(data); err != nil {
            return 0, nil, err
        }
        return kind, data, nil
    }
    return kind, nil, nil
}

func joinStrings(strs []string, sep string) string {
    if len(strs) == 0 {
        return ""
    }
    out := strs[0]
    for _, s := range strs[1:] {
        out += sep + s
    }
    return out
}

// Pool manages a pool of connections (add this at the end of the file)
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
    select {
    case conn := <-p.conns:
        return conn, nil
    default:
        return Dial()
    }
}

func (p *Pool) Release(conn *Client) error {
    select {
    case p.conns <- conn:
        return nil
    default:
        return conn.Close()
    }
}

func (p *Pool) Close() error {
    close(p.conns)
    for conn := range p.conns {
        conn.Close()
    }
    return nil
}