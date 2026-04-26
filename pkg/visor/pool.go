// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import "sync"

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
