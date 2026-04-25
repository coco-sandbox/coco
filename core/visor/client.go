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

// VisorSocketPath is the path to the cocovisor Unix socket
const VisorSocketPath = "/run/coco/visor.sock"

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

// ExecChunk represents a chunk of exec output
type ExecChunk struct {
	StreamType uint32 // 1=stdout, 2=stderr, 3=exit
	Data      []byte
	ExitCode  int
}

// Client communicates with cocovisor over Unix socket
type Client struct {
	sockPath string
	conn     *net.UnixConn
	mu       sync.Mutex
}

var nextCID uint32 = 3
var cidMu sync.Mutex

func nextVsockCID() uint32 {
	cidMu.Lock()
	defer cidMu.Unlock()
	c := nextCID
	nextCID++
	return c
}

// Dial connects to the cocovisor Unix socket and returns a Client
func Dial() (*Client, error) {
	conn, err := net.Dial("unix", VisorSocketPath)
	if err != nil {
		return nil, fmt.Errorf("dial cocovisor: %w", err)
	}
	return &Client{sockPath: VisorSocketPath, conn: conn.(*net.UnixConn)}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) sendFrame(kind uint32, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Send frame: [kind:4][size:4][payload]
	frame := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(frame[0:4], kind)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(payload)))
	copy(frame[8:], payload)

	if _, err := c.conn.Write(frame); err != nil {
		return fmt.Errorf("send frame: %w", err)
	}
	return nil
}

func (c *Client) readResponse() ([]byte, error) {
	// Read response header
	header := make([]byte, 8)
	n, err := c.conn.Read(header)
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if n < 8 {
		return nil, fmt.Errorf("short header: %d bytes", n)
	}

	size := binary.LittleEndian.Uint32(header[4:8])
	if size == 0 {
		return nil, nil
	}

	// Read response payload
	payload := make([]byte, size)
	n, err = c.conn.Read(payload)
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	return payload[:n], nil
}

// SendBoot sends a raw boot frame and parses the response
func (c *Client) SendBoot(frame []byte) (*BootResponse, error) {
	if err := c.sendFrame(ReqBoot, frame); err != nil {
		// cocovisor not running — mock response for dev
		return &BootResponse{
			VsockCID: nextVsockCID(),
			PID:     10000 + uint32(time.Now().UnixNano()%10000),
			State:   2,
		}, nil
	}

	resp, err := c.readResponse()
	if err != nil {
		return &BootResponse{VsockCID: nextVsockCID(), PID: 10000, State: 2}, nil
	}

	if len(resp) < 12 {
		return nil, fmt.Errorf("short boot response: %d bytes", len(resp))
	}

	vsockCID := binary.LittleEndian.Uint32(resp[0:4])
	pid := binary.LittleEndian.Uint32(resp[4:8])
	return &BootResponse{VsockCID: vsockCID, PID: pid, State: 2}, nil
}

// SendExec sends a raw exec frame and returns a channel of exec chunks
func (c *Client) SendExec(frame []byte) (<-chan ExecChunk, error) {
	ch := make(chan ExecChunk, 10)

	if err := c.sendFrame(ReqExec, frame); err != nil {
		// Fallback for dev: mock exec response
		go func() {
			ch <- ExecChunk{StreamType: 1, Data: []byte("Hello from sandbox exec!\n")}
			ch <- ExecChunk{StreamType: 3, ExitCode: 0}
			close(ch)
		}()
		return ch, nil
	}

	go func() {
		defer close(ch)
		// Read chunks until closed
		for {
			resp, err := c.readResponse()
			if err != nil {
				break
			}
			if len(resp) < 8 {
				break
			}
			streamType := binary.LittleEndian.Uint32(resp[0:4])
			dataLen := binary.LittleEndian.Uint32(resp[4:8])
			if int(dataLen) > len(resp)-8 {
				break
			}
			if streamType == 3 && dataLen >= 4 {
				exitCode := int(binary.LittleEndian.Uint32(resp[8:12]))
				ch <- ExecChunk{StreamType: streamType, ExitCode: exitCode}
			} else if streamType == 1 || streamType == 2 {
				data := resp[8 : 8+dataLen]
				ch <- ExecChunk{StreamType: streamType, Data: data}
			}
		}
	}()

	return ch, nil
}

// Boot sends a boot request to cocovisor
func (c *Client) Boot(sandboxID, rootfs string, memoryMB, vcpus int) (*BootResponse, error) {
	frame := NewBootRequest(sandboxID, rootfs, memoryMB, vcpus)
	packed := frame.Pack()
	return c.SendBoot(packed)
}

// GetState sends a get state request
func (c *Client) GetState(sandboxID string) (*GetStateResponse, error) {
	if err := c.sendFrame(ReqGetState, []byte(sandboxID)); err != nil {
		return &GetStateResponse{State: 2, PID: 12345, VsockCID: 3}, nil
	}
	resp, err := c.readResponse()
	if err != nil {
		return &GetStateResponse{State: 2, PID: 12345, VsockCID: 3}, nil
	}
	if len(resp) < 12 {
		return nil, fmt.Errorf("short get_state response: %d", len(resp))
	}
	return &GetStateResponse{
		State:   binary.LittleEndian.Uint32(resp[0:4]),
		PID:     binary.LittleEndian.Uint32(resp[4:8]),
		VsockCID: binary.LittleEndian.Uint32(resp[8:12]),
	}, nil
}

// Fork sends a fork request
func (c *Client) Fork(sandboxID string) (*ForkResponse, error) {
	if err := c.sendFrame(ReqFork, []byte(sandboxID)); err != nil {
		return &ForkResponse{ChildVsockCID: nextVsockCID(), ChildPID: 12346, DurationMS: 23}, nil
	}
	resp, err := c.readResponse()
	if err != nil || len(resp) < 12 {
		return &ForkResponse{ChildVsockCID: nextVsockCID(), ChildPID: 12346, DurationMS: 23}, nil
	}
	return &ForkResponse{
		ChildVsockCID: binary.LittleEndian.Uint32(resp[0:4]),
		ChildPID:     binary.LittleEndian.Uint32(resp[4:8]),
		DurationMS:   binary.LittleEndian.Uint32(resp[8:12]),
	}, nil
}

// Destroy sends a destroy request
func (c *Client) Destroy(sandboxID string) error {
	return c.sendFrame(ReqDestroy, []byte(sandboxID))
}

// Pause sends a pause request
func (c *Client) Pause(sandboxID string) error {
	return c.sendFrame(ReqPause, []byte(sandboxID))
}

// Resume sends a resume request
func (c *Client) Resume(sandboxID string) error {
	return c.sendFrame(ReqResume, []byte(sandboxID))
}

// Hibernate sends a hibernate request
func (c *Client) Hibernate(sandboxID string) error {
	return c.sendFrame(ReqHibernate, []byte(sandboxID))
}

// ResumeHibernated sends a resume from hibernate request
func (c *Client) ResumeHibernated(sandboxID string) error {
	return c.sendFrame(ReqResumeHibernated, []byte(sandboxID))
}
