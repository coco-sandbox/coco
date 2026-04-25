// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// ErrShortFrame is returned when a frame is too short to contain expected data.
var ErrShortFrame = errors.New("short frame")

// ErrUnexpectedKind is returned when the frame kind does not match expectations.
var ErrUnexpectedKind = errors.New("unexpected frame kind")

// ErrConnectionFailed is returned when the socket connection cannot be established.
var ErrConnectionFailed = errors.New("visor socket connection failed")

// Client is a connection to the cocovisor Unix socket.
type Client struct {
	conn net.Conn
}

// Dial connects to the visor socket.
func Dial() (*Client, error) {
	conn, err := net.Dial("unix", SockPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	return &Client{conn: conn}, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendBoot sends a Boot request and waits for the response.
func (c *Client) SendBoot(frame []byte) (*BootResponse, error) {
	if _, err := c.conn.Write(frame); err != nil {
		return nil, fmt.Errorf("write boot frame: %w", err)
	}

	// Read response frame header
	header := make([]byte, 8)
	if _, err := readFull(c.conn, header); err != nil {
		return nil, fmt.Errorf("read boot response header: %w", err)
	}

	size := binary.LittleEndian.Uint32(header[4:8])
	payload := make([]byte, size)
	if _, err := readFull(c.conn, payload); err != nil {
		return nil, fmt.Errorf("read boot response payload: %w", err)
	}

	resp, err := ParseBootResponse(append(header, payload...))
	if err != nil {
		return nil, fmt.Errorf("parse boot response: %w", err)
	}
	return resp, nil
}

// SendExec sends an Exec request and returns a channel of response chunks.
// The caller must drain the channel to completion. The channel closes when done.
func (c *Client) SendExec(frame []byte) (<-chan *ExecChunk, error) {
	if _, err := c.conn.Write(frame); err != nil {
		return nil, fmt.Errorf("write exec frame: %w", err)
	}

	chunks := make(chan *ExecChunk)

	go func() {
		defer close(chunks)
		for {
			header := make([]byte, 8)
			_, err := readFull(c.conn, header)
			if err != nil {
				return
			}

			size := binary.LittleEndian.Uint32(header[4:8])
			payload := make([]byte, size)
			_, err = readFull(c.conn, payload)
			if err != nil {
				return
			}

			fullFrame := append(header, payload...)
			chunk, err := ReadExecChunk(fullFrame)
			if err != nil {
				return
			}

			chunks <- chunk
			if chunk.StreamType == 3 {
				// Exit chunk — end of stream
				return
			}
		}
	}()

	return chunks, nil
}

// readFull reads exactly n bytes from conn, handling partial reads.
func readFull(conn net.Conn, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		got, err := conn.Read(buf[n:])
		if err != nil {
			return n, err
		}
		n += got
	}
	return n, nil
}
