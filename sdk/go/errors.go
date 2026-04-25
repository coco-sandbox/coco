// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package coco

import "fmt"

// Error codes matching the API spec
const (
	ErrCodeSandboxNotFound      = "SANDBOX_NOT_FOUND"
	ErrCodeSandboxAlreadyRunning = "SANDBOX_ALREADY_RUNNING"
	ErrCodeSandboxNotRunning    = "SANDBOX_NOT_RUNNING"
	ErrCodeSandboxHibernated    = "SANDBOX_HIBERNATED"
	ErrCodeCheckpointNotFound    = "CHECKPOINT_NOT_FOUND"
	ErrCodeCheckpointChainBroken = "CHECKPOINT_CHAIN_BROKEN"
	ErrCodeVsockCIDExhausted    = "VSOCK_CID_EXHAUSTED"
	ErrCodeExecTimeout          = "EXEC_TIMEOUT"
	ErrCodeExecFailed           = "EXEC_FAILED"
	ErrCodeHypervisorError      = "HYPERVISOR_ERROR"
	ErrCodeNetworkError         = "NETWORK_ERROR"
	ErrCodeEBPFError           = "EBPF_ERROR"
	ErrCodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
)

// Error represents an API error response
type Error struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// SDKError represents an error from the SDK itself
type SDKError struct {
	Message string
	Err    error
}

func (e *SDKError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *SDKError) Unwrap() error {
	return e.Err
}

// ErrNotFound is returned when a resource is not found
var ErrNotFound = &SDKError{Message: "resource not found"}

// ErrTimeout is returned when an operation times out
var ErrTimeout = &SDKError{Message: "operation timed out"}

// ErrConnection is returned when the connection to coco-core fails
var ErrConnection = &SDKError{Message: "failed to connect to coco-core"}
