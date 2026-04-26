// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	AuditEventAuthSuccess          AuditEventType = "AUTH_SUCCESS"
	AuditEventAuthFailure          AuditEventType = "AUTH_FAILURE"
	AuditEventAuthorizationFailure AuditEventType = "AUTHZ_FAILURE"
	AuditEventSandboxCreate        AuditEventType = "SANDBOX_CREATE"
	AuditEventSandboxDelete        AuditEventType = "SANDBOX_DELETE"
	AuditEventConfigChange         AuditEventType = "CONFIG_CHANGE"
)

// AuditLogEntry is a structured audit log entry per spec.
type AuditLogEntry struct {
	Timestamp string         `json:"timestamp"`
	EventType AuditEventType `json:"event_type"`
	User      string         `json:"user,omitempty"`
	SourceIP  string         `json:"source_ip,omitempty"`
	Outcome   string         `json:"outcome"`
	Details   string         `json:"details,omitempty"`
	SandboxID string         `json:"sandbox_id,omitempty"`
}

// AuditLogger handles structured audit logging.
type AuditLogger struct {
	mu        sync.RWMutex
	component string
}

func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		component: "coco-gateway",
	}
}

// Log emits a structured JSON audit log entry.
func (l *AuditLogger) Log(entry AuditLogEntry) {
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	entry.SourceIP = "unknown"
	if j, err := json.Marshal(entry); err == nil {
		log.Printf("[AUDIT] %s", string(j))
	}
}

// LogAuthSuccess logs a successful authentication event.
func (l *AuditLogger) LogAuthSuccess(user, sourceIP string) {
	l.Log(AuditLogEntry{
		EventType: AuditEventAuthSuccess,
		User:      user,
		SourceIP:  sourceIP,
		Outcome:   "success",
	})
}

// LogAuthFailure logs a failed authentication attempt.
func (l *AuditLogger) LogAuthFailure(user, sourceIP, reason string) {
	l.Log(AuditLogEntry{
		EventType: AuditEventAuthFailure,
		User:      user,
		SourceIP:  sourceIP,
		Outcome:   "failure",
		Details:   reason,
	})
}

// LogAuthzFailure logs an authorization failure.
func (l *AuditLogger) LogAuthzFailure(user, sourceIP, resource, reason string) {
	l.Log(AuditLogEntry{
		EventType: AuditEventAuthorizationFailure,
		User:      user,
		SourceIP:  sourceIP,
		Outcome:   "denied",
		Details:   resource + ": " + reason,
	})
}

// LogSandboxCreate logs sandbox creation.
func (l *AuditLogger) LogSandboxCreate(user, sourceIP, sandboxID, templateID string) {
	l.Log(AuditLogEntry{
		EventType: AuditEventSandboxCreate,
		User:      user,
		SourceIP:  sourceIP,
		Outcome:   "created",
		SandboxID: sandboxID,
		Details:   "template=" + templateID,
	})
}

// LogSandboxDelete logs sandbox deletion.
func (l *AuditLogger) LogSandboxDelete(user, sourceIP, sandboxID string) {
	l.Log(AuditLogEntry{
		EventType: AuditEventSandboxDelete,
		User:      user,
		SourceIP:  sourceIP,
		Outcome:   "deleted",
		SandboxID: sandboxID,
	})
}

// LogConfigChange logs security policy configuration changes.
func (l *AuditLogger) LogConfigChange(user, sourceIP, configType, change string) {
	l.Log(AuditLogEntry{
		EventType: AuditEventConfigChange,
		User:      user,
		SourceIP:  sourceIP,
		Outcome:   "changed",
		Details:   configType + ": " + change,
	})
}

// GetClientIP extracts the client IP from a request.
func GetClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return real
	}
	return r.RemoteAddr
}
