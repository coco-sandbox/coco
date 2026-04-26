// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	v1connect "coco/pkg/api/v1/v1connect"
)

// Client is a Coco API client
type Client struct {
	gateway v1connect.GatewayServiceClient
	node    v1connect.NodeServiceClient
}

// NewClient creates a new Coco API client
func NewClient(addr string) (*Client, error) {
	if addr == "" {
		addr = "http://localhost:9090"
	}

	return &Client{
		gateway: v1connect.NewGatewayServiceClient(http.DefaultClient, addr),
		node:    v1connect.NewNodeServiceClient(http.DefaultClient, addr),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return nil
}

// Gateway returns the Gateway service client
func (c *Client) Gateway() v1connect.GatewayServiceClient {
	return c.gateway
}

// Node returns the Node service client
func (c *Client) Node() v1connect.NodeServiceClient {
	return c.node
}

// WaitForReady waits for the connection to be ready
func (c *Client) WaitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return fmt.Errorf("connection not ready after %v", timeout)
	case <-time.After(timeout):
		return nil
	}
}
