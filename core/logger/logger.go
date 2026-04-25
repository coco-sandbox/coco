// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents log level priority
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Record is a structured log record
type Record struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Component string         `json:"component"`
	Message   string         `json:"message"`
	TraceID   string         `json:"trace_id,omitempty"`
	SandboxID string         `json:"sandbox_id,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// Logger provides structured JSON logging
type Logger struct {
	mu        sync.Mutex
	output    io.Writer
	level     Level
	component string
}

// New creates a new Logger for a component
func New(component string) *Logger {
	return &Logger{
		output:    os.Stdout,
		level:     INFO,
		component: component,
	}
}

// SetOutput sets the destination writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithTraceID returns a logger with trace_id field
func (l *Logger) WithTraceID(traceID string) *Logger {
	return &Logger{
		output:    l.output,
		level:     l.level,
		component: l.component,
	}
}

// WithSandboxID returns a logger with sandbox_id field
func (l *Logger) WithSandboxID(sandboxID string) *Logger {
	return &Logger{
		output:    l.output,
		level:     l.level,
		component: l.component,
	}
}

func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	rec := Record{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Component: l.component,
		Message:   msg,
		Fields:    fields,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: marshal failed: %v\n", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.output, string(data))
}

// Debug logs at DEBUG level
func (l *Logger) Debug(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(DEBUG, msg, f)
}

// Info logs at INFO level
func (l *Logger) Info(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(INFO, msg, f)
}

// Warn logs at WARN level
func (l *Logger) Warn(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(WARN, msg, f)
}

// Error logs at ERROR level
func (l *Logger) Error(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(ERROR, msg, f)
}

// Fatal logs at ERROR level and exits
func (l *Logger) Fatal(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(ERROR, msg, f)
	os.Exit(1)
}

func mergeFields(fields []map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]any)
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// Global default logger for core component
var coreLog = New("coco-core")

// Package-level helpers
func Info(msg string, fields ...map[string]any) { coreLog.Info(msg, fields...) }
func Warn(msg string, fields ...map[string]any) { coreLog.Warn(msg, fields...) }
func Error(msg string, fields ...map[string]any) { coreLog.Error(msg, fields...) }
func Debug(msg string, fields ...map[string]any) { coreLog.Debug(msg, fields...) }

// ForComponent creates a logger for a specific component
func ForComponent(name string) *Logger {
	return New(name)
}
