// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionCreate       AuditAction = "create"
	AuditActionRead         AuditAction = "read"
	AuditActionUpdate       AuditAction = "update"
	AuditActionDelete       AuditAction = "delete"
	AuditActionExec         AuditAction = "exec"
	AuditActionFork         AuditAction = "fork"
	AuditActionHibernate    AuditAction = "hibernate"
	AuditActionResume       AuditAction = "resume"
	AuditActionCheckpoint   AuditAction = "checkpoint"
	AuditActionReplay       AuditAction = "replay"
	AuditActionAuthSuccess  AuditAction = "auth_success"
	AuditActionAuthFailure  AuditAction = "auth_failure"
	AuditActionRateLimit    AuditAction = "rate_limit"
)

// AuditResult represents the outcome of an audited action
type AuditResult string

const (
	AuditResultSuccess AuditResult = "success"
	AuditResultFailure AuditResult = "failure"
	AuditResultDenied  AuditResult = "denied"
)

// AuditEntry represents a structured audit log entry
type AuditEntry struct {
	// Timestamp when the action occurred (RFC3339 format)
	Timestamp string `json:"timestamp"`

	// Unique identifier for this audit entry
	EntryID string `json:"entry_id"`

	// Action performed
	Action AuditAction `json:"action"`

	// Result of the action
	Result AuditResult `json:"result"`

	// Resource being acted upon
	ResourceType string `json:"resource_type"` // e.g., "sandbox", "checkpoint", "config"
	ResourceID   string `json:"resource_id,omitempty"`

	// Actor information (who performed the action)
	ActorType string `json:"actor_type"` // e.g., "user", "api_key", "system"
	ActorID   string `json:"actor_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`

	// Request context
	RequestID    string `json:"request_id,omitempty"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent,omitempty"`
	AuthMethod   string `json:"auth_method,omitempty"`

	// Action details
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// Error information (if result != success)
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Integrity
	Checksum string `json:"checksum,omitempty"`
}

// AuditLogger handles audit logging operations
type AuditLogger struct {
	logger *log.Logger
	store  *AuditStore
}

// AuditStore handles persistence of audit entries
type AuditStore struct {
	// In production this would be a append-only log store
	// with cryptographic chaining for tamper evidence
	entries []*AuditEntry
	mu      int // Using sync.Mutex in practice
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *log.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
		store:  &AuditStore{},
	}
}

// Log writes an audit entry
func (a *AuditLogger) Log(entry *AuditEntry) error {
	// Set timestamp if not provided
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	// Generate entry ID if not provided
	if entry.EntryID == "" {
		entry.EntryID = generateAuditID(entry)
	}

	// Compute checksum for integrity
	entry.Checksum = computeAuditChecksum(entry)

	// Write to log (JSON format for machine parsing)
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	a.logger.Printf("[AUDIT] %s", string(data))

	// In production: write to append-only audit store
	// a.store.Append(entry)

	return nil
}

// LogFromRequest creates an audit entry from HTTP request context
func (a *AuditLogger) LogFromRequest(
	action AuditAction,
	result AuditResult,
	r *http.Request,
	resourceType, resourceID string,
	metadata map[string]string,
) error {
	entry := &AuditEntry{
		Action:       action,
		Result:       result,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ClientIP:     clientIPFromRequest(r),
		UserAgent:    r.UserAgent(),
		Metadata:     metadata,
	}

	// Extract request ID if present
	if reqID := r.Context().Value(contextKeyRequestID); reqID != nil {
		entry.RequestID = reqID.(string)
	}

	// Extract auth context if present
	if apiKey := APIKeyFromContext(r.Context()); apiKey != nil {
		entry.ActorType = "api_key"
		entry.ActorID = apiKey.Key
		entry.TenantID = apiKey.TenantID
		entry.AuthMethod = "api_key"
	}

	return a.Log(entry)
}

// LogSandboxAction is a helper for common sandbox actions
func (a *AuditLogger) LogSandboxAction(
	action AuditAction,
	result AuditResult,
	sandboxID string,
	r *http.Request,
	err error,
) error {
	metadata := make(map[string]string)
	if err != nil {
		metadata["error"] = err.Error()
	}

	entry := &AuditEntry{
		Action:       action,
		Result:       result,
		ResourceType: "sandbox",
		ResourceID:   sandboxID,
		ClientIP:     clientIPFromRequest(r),
		UserAgent:    r.UserAgent(),
		Metadata:     metadata,
		Description:  fmt.Sprintf("sandbox %s: %s", action, result),
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
	}

	return a.Log(entry)
}

// LogAuthAction logs authentication events
func (a *AuditLogger) LogAuthAction(
	result AuditResult,
	r *http.Request,
	authMethod string,
	err error,
) error {
	metadata := make(map[string]string)
	if err != nil {
		metadata["error"] = err.Error()
	}

	entry := &AuditEntry{
		Action:       AuditActionAuthSuccess,
		Result:       result,
		ResourceType: "auth",
		ClientIP:     clientIPFromRequest(r),
		UserAgent:    r.UserAgent(),
		AuthMethod:   authMethod,
		Metadata:     metadata,
	}

	if result == AuditResultFailure {
		entry.Action = AuditActionAuthFailure
	}

	return a.Log(entry)
}

// LogRateLimitAction logs rate limit events
func (a *AuditLogger) LogRateLimitAction(
	r *http.Request,
	tenantID string,
) error {
	entry := &AuditEntry{
		Action:       AuditActionRateLimit,
		Result:       AuditResultDenied,
		ResourceType: "rate_limit",
		ClientIP:     clientIPFromRequest(r),
		UserAgent:    r.UserAgent(),
		TenantID:     tenantID,
		Description:  "rate limit exceeded",
	}

	return a.Log(entry)
}

// generateAuditID creates a unique ID for an audit entry
func generateAuditID(entry *AuditEntry) string {
	// In production: use a distributed ID generator
	data := fmt.Sprintf("%s:%s:%s:%d", entry.Action, entry.ResourceType, entry.ResourceID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("aud_%x", hash[:16])
}

// computeAuditChecksum computes a checksum for audit entry integrity
func computeAuditChecksum(entry *AuditEntry) string {
	// Create a copy without the checksum field
	entryCopy := *entry
	entryCopy.Checksum = ""

	data, _ := json.Marshal(entryCopy)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash)
}

// clientIPFromRequest extracts client IP considering proxies
func clientIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Context keys for request context values
type contextKeyType string

const (
	contextKeyRequestID contextKeyType = "request_id"
	contextKeyAuth     contextKeyType = "auth"
)

// ContextWithRequestID returns a context with the request ID
func ContextWithRequestID(ctx interface{ Value(any) any }, requestID string) interface{} {
	// This would be implemented with context.WithValue in practice
	return ctx
}
