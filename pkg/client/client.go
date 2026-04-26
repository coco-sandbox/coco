// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package client

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/coco-sandbox/coco/pkg/api/v1"
)

// Client is a Coco API client
type Client struct {
	conn    *grpc.ClientConn
	gateway v1.GatewayServiceClient
	node    v1.NodeServiceClient
}

// NewClient creates a new Coco API client
func NewClient(addr string) (*Client, error) {
	if addr == "" {
		addr = "localhost:9090"
	}

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}

	return &Client{
		conn:    conn,
		gateway: v1.NewGatewayServiceClient(conn),
		node:    v1.NewNodeServiceClient(conn),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Gateway returns the Gateway service client
func (c *Client) Gateway() v1.GatewayServiceClient {
	return c.gateway
}

// Node returns the Node service client
func (c *Client) Node() v1.NodeServiceClient {
	return c.node
}

// WaitForReady waits for the connection to be ready
func (c *Client) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.conn.GetState() == net.StatusReady {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("connection not ready after %v", timeout)
}
