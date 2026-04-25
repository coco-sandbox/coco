// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package coco

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the main SDK client for interacting with coco-core
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	timeout    time.Duration
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithAddress sets the address of the coco-core server
func WithAddress(addr string) ClientOption {
	return func(c *Client) {
		c.baseURL = fmt.Sprintf("http://%s", addr)
	}
}

// WithAPIKey sets the API key for authentication
func WithAPIKey(key string) ClientOption {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithTimeout sets the default timeout for requests
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// NewClient creates a new Coco SDK client
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		baseURL: "http://localhost:4747",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// doRequest performs an HTTP request with the given method, path, and body
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, &SDKError{Message: "failed to marshal request body", Err: err}
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, &SDKError{Message: "failed to create request", Err: err}
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	return c.httpClient.Do(req)
}

// parseResponse parses the response body into the given type
func (c *Client) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error Error `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return &SDKError{Message: fmt.Sprintf("request failed with status %d", resp.StatusCode)}
		}
		return &errResp.Error
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return &SDKError{Message: "failed to parse response", Err: err}
		}
	}

	return nil
}

// Health checks the health of the coco-core server
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return nil, err
	}

	var result HealthResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Ready checks if coco-core is ready to serve requests
func (c *Client) Ready(ctx context.Context) (*ReadyResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/ready", nil)
	if err != nil {
		return nil, err
	}

	var result ReadyResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateSandbox creates a new sandbox
func (c *Client) CreateSandbox(ctx context.Context, config *SandboxConfig) (*CreateSandboxResponse, error) {
	if config == nil {
		config = &SandboxConfig{}
	}

	// Apply defaults
	if config.Template == "" {
		config.Template = "alpine"
	}
	if config.MemoryMB == 0 {
		config.MemoryMB = 512
	}
	if config.VCPUs == 0 {
		config.VCPUs = 2
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/v1/sandboxes", config)
	if err != nil {
		return nil, err
	}

	var result CreateSandboxResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSandbox gets a sandbox by ID
func (c *Client) GetSandbox(ctx context.Context, id string) (*Sandbox, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/sandboxes/%s", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Sandbox *Sandbox `json:"sandbox"`
	}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Sandbox, nil
}

// ListSandboxes lists all sandboxes with optional filtering
func (c *Client) ListSandboxes(ctx context.Context, opts ...ListOption) (*ListSandboxesResponse, error) {
	params := url.Values{}
	for _, opt := range opts {
		opt.apply(params)
	}

	path := "/v1/sandboxes"
	if query := params.Encode(); query != "" {
		path = path + "?" + query
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result ListSandboxesResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListOption is a functional option for list operations
type ListOption interface {
	apply(v url.Values)
}

type listOptionFunc func(url.Values)

func (f listOptionFunc) apply(v url.Values) { f(v) }

// WithStateFilter filters sandboxes by state
func WithStateFilter(state SandboxState) ListOption {
	return listOptionFunc(func(v url.Values) {
		v.Set("state", string(state))
	})
}

// WithLabelFilter filters sandboxes by label
func WithLabelFilter(key, value string) ListOption {
	return listOptionFunc(func(v url.Values) {
		v.Set("label_key", key)
		v.Set("label_value", value)
	})
}

// WithOffset sets the offset for pagination
func WithOffset(offset int) ListOption {
	return listOptionFunc(func(v url.Values) {
		v.Set("offset", fmt.Sprintf("%d", offset))
	})
}

// WithLimit sets the limit for pagination
func WithLimit(limit int) ListOption {
	return listOptionFunc(func(v url.Values) {
		v.Set("limit", fmt.Sprintf("%d", limit))
	})
}

// DestroySandbox destroys a sandbox by ID
func (c *Client) DestroySandbox(ctx context.Context, id string) (*DestroyResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/v1/sandboxes/%s", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var result DestroyResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ForkSandbox forks a sandbox
func (c *Client) ForkSandbox(ctx context.Context, id string, req *ForkRequest) (*ForkResponse, error) {
	if req == nil {
		req = &ForkRequest{}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/fork", url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}

	var result ForkResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// HibernateSandbox hibernates a sandbox
func (c *Client) HibernateSandbox(ctx context.Context, id string) (*HibernateResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/hibernate", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var result HibernateResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ResumeSandbox resumes a hibernated sandbox
func (c *Client) ResumeSandbox(ctx context.Context, id string) (*Sandbox, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/resume", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID    string        `json:"id"`
		State SandboxState `json:"state"`
	}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &Sandbox{ID: result.ID, State: result.State}, nil
}

// PauseSandbox pauses a running sandbox
func (c *Client) PauseSandbox(ctx context.Context, id string) (*Sandbox, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/pause", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID    string        `json:"id"`
		State SandboxState `json:"state"`
	}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &Sandbox{ID: result.ID, State: result.State}, nil
}

// Exec executes a command in a sandbox and streams the output
func (c *Client) Exec(ctx context.Context, id string, req *ExecRequest, handler func(*ExecChunk) error) error {
	if req == nil {
		req = &ExecRequest{}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/exec", url.PathEscape(id)), req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error Error `json:"error"`
		}
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr != nil {
			return &SDKError{Message: fmt.Sprintf("exec failed with status %d", resp.StatusCode)}
		}
		return &errResp.Error
	}

	// Streaming response - decode SSE format
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk ExecChunk
		// Read the next JSON object from the stream
		// The response is in SSE format: data: {...}\n\n
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return &SDKError{Message: "failed to read exec stream", Err: err}
		}

		// Skip "data:" prefix
		if delim, ok := token.(json.Delim); ok && delim == 'd' {
			var data struct {
				Type    string `json:"type"`
				Data    string `json:"data,omitempty"`
				ExitCode int   `json:"exit_code,omitempty"`
			}
			if err := decoder.Decode(&data); err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			chunk.Type = data.Type
			chunk.Data = data.Data
			chunk.ExitCode = data.ExitCode
		} else {
			continue
		}

		if handler != nil {
			if err := handler(&chunk); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExecSync executes a command synchronously and returns all output
func (c *Client) ExecSync(ctx context.Context, id string, req *ExecRequest) (string, string, int, error) {
	var stdout, stderr string
	var exitCode int

	err := c.Exec(ctx, id, req, func(chunk *ExecChunk) error {
		switch chunk.Type {
		case "stdout":
			stdout += chunk.Data
		case "stderr":
			stderr += chunk.Data
		case "exit":
			exitCode = chunk.ExitCode
		}
		return nil
	})

	if err != nil {
		return stdout, stderr, exitCode, err
	}

	return stdout, stderr, exitCode, nil
}

// CreateCheckpoint creates a checkpoint for a sandbox
func (c *Client) CreateCheckpoint(ctx context.Context, id, name, description string) (*Checkpoint, error) {
	req := struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}{Name: name, Description: description}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/checkpoints", url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}

	var result Checkpoint
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListCheckpoints lists all checkpoints for a sandbox
func (c *Client) ListCheckpoints(ctx context.Context, id string) ([]*Checkpoint, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/sandboxes/%s/checkpoints", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []*Checkpoint `json:"items"`
	}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// UndoToCheckpoint undoes a sandbox to a specific checkpoint
func (c *Client) UndoToCheckpoint(ctx context.Context, id, checkpointID string) error {
	req := struct {
		Checkpoint string `json:"checkpoint"`
	}{Checkpoint: checkpointID}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sandboxes/%s/undo", url.PathEscape(id)), req)
	if err != nil {
		return err
	}

	return c.parseResponse(resp, nil)
}

// Metrics fetches Prometheus metrics
func (c *Client) Metrics(ctx context.Context) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/metrics", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &SDKError{Message: "failed to read metrics", Err: err}
	}

	return string(data), nil
}

// UpdateSandbox updates a sandbox's name or labels
func (c *Client) UpdateSandbox(ctx context.Context, id string, updates *SandboxConfig) (*Sandbox, error) {
	req := struct {
		Name   string            `json:"name,omitempty"`
		Labels map[string]string `json:"labels,omitempty"`
	}{
		Name:   updates.Name,
		Labels: updates.Labels,
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/v1/sandboxes/%s", url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Sandbox *Sandbox `json:"sandbox"`
	}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Sandbox, nil
}

// String returns a string representation of the sandbox state
func (s SandboxState) String() string {
	return string(s)
}

// IsTerminal returns true if the state is a terminal state
func (s SandboxState) IsTerminal() bool {
	return s == SandboxStateStopped || s == SandboxStateError
}

// IsActive returns true if the sandbox is in an active state
func (s SandboxState) IsActive() bool {
	return s == SandboxStateRunning || s == SandboxStateCreating
}
