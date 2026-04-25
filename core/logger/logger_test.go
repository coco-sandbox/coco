// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package logger

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestRecordJSON(t *testing.T) {
	rec := Record{
		TS:        "2026-04-26T12:00:00Z",
		Level:     "INFO",
		Component: "test",
		Message:   "hello",
		TraceID:   "trace-123",
		Fields:    Fields{"key": "value"},
	}

	b := rec.JSON()

	var parsed Record
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if parsed.TS != rec.TS {
		t.Errorf("TS = %v, want %v", parsed.TS, rec.TS)
	}
	if parsed.Level != rec.Level {
		t.Errorf("Level = %v, want %v", parsed.Level, rec.Level)
	}
	if parsed.Component != rec.Component {
		t.Errorf("Component = %v, want %v", parsed.Component, rec.Component)
	}
	if parsed.Message != rec.Message {
		t.Errorf("Message = %v, want %v", parsed.Message, rec.Message)
	}
	if parsed.TraceID != rec.TraceID {
		t.Errorf("TraceID = %v, want %v", parsed.TraceID, rec.TraceID)
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer

	l := New("test", &buf)

	l.SetLevel(INFO)

	// Debug should be filtered
	l.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("DEBUG should be filtered when level is INFO")
	}

	// Info should pass
	buf.Reset()
	l.Info("info message")
	if buf.Len() == 0 {
		t.Error("INFO should be logged")
	}

	// Warn should pass
	buf.Reset()
	l.Warn("warn message")
	if buf.Len() == 0 {
		t.Error("WARN should be logged")
	}

	// Error should pass
	buf.Reset()
	l.Error("error message")
	if buf.Len() == 0 {
		t.Error("ERROR should be logged")
	}
}

func TestLoggerTraceID(t *testing.T) {
	var buf bytes.Buffer

	l := New("test", &buf)

	l.SetTraceID("my-trace-id")
	l.Info("test message")

	var rec Record
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if rec.TraceID != "my-trace-id" {
		t.Errorf("TraceID = %v, want %v", rec.TraceID, "my-trace-id")
	}
}

func TestLoggerComponent(t *testing.T) {
	var buf bytes.Buffer

	l := New("mycomponent", &buf)
	l.Info("test message")

	var rec Record
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if rec.Component != "mycomponent" {
		t.Errorf("Component = %v, want %v", rec.Component, "mycomponent")
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer

	l := New("test", &buf)
	l.Info("with fields", Fields{"count": 42, "name": "foo"})

	var rec Record
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if rec.Fields["count"] != 42 {
		t.Errorf("Fields[count] = %v, want 42", rec.Fields["count"])
	}
	if rec.Fields["name"] != "foo" {
		t.Errorf("Fields[name] = %v, want foo", rec.Fields["name"])
	}
}

func TestNewStdOut(t *testing.T) {
	l := NewStdOut("test")
	if l.component != "test" {
		t.Errorf("component = %v, want test", l.component)
	}
	if l.output == nil {
		t.Error("output should not be nil")
	}
}

func TestWithFields(t *testing.T) {
	l := New("test", &bytes.Buffer{})
	l.SetTraceID("trace-1")

	l2 := l.WithFields(Fields{"extra": "value"})

	if l2.traceID != "trace-1" {
		t.Errorf("WithFields should preserve traceID, got %v", l2.traceID)
	}
	if l2.component != "test" {
		t.Errorf("WithFields should preserve component, got %v", l2.component)
	}
}
