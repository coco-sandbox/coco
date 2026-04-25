// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// Package metrics provides metrics collection for the Coco API server.
package metrics

import (
	"sync"
	"time"
)

// Metrics holds all metrics counters and histograms
type Metrics struct {
	mu                  sync.RWMutex
	SandboxCountByState map[string]int
	CreatesTotal        map[string]int
	CreateDuration      []float64
	DestroysTotal      int
	ExecDuration        []float64
	ForkDuration        []float64
	HibernateDuration   []float64
	HibernateSizeBytes int64
	MemoryUsedBytes    int64
	CPUSecondsTotal    float64
	NetworkBytesIngress int64
	NetworkBytesEgress  int64
}

// New creates a new Metrics instance
func New() *Metrics {
	return &Metrics{
		SandboxCountByState: make(map[string]int),
		CreatesTotal:        make(map[string]int),
		CreateDuration:      make([]float64, 0, 100),
		ExecDuration:        make([]float64, 0, 100),
		ForkDuration:        make([]float64, 0, 100),
		HibernateDuration:   make([]float64, 0, 100),
	}
}

// RecordCreate records a sandbox creation
func (m *Metrics) RecordCreate(template string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreatesTotal[template]++
	m.CreateDuration = append(m.CreateDuration, float64(d.Milliseconds()))
	m.SandboxCountByState["creating"]++
	m.SandboxCountByState["running"]++
}

// RecordDestroy records a sandbox destruction
func (m *Metrics) RecordDestroy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DestroysTotal++
	if c, ok := m.SandboxCountByState["running"]; ok && c > 0 {
		m.SandboxCountByState["running"]--
	}
	m.SandboxCountByState["destroyed"]++
}

// RecordExec records exec latency
func (m *Metrics) RecordExec(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecDuration = append(m.ExecDuration, float64(d.Milliseconds()))
}

// RecordFork records fork latency
func (m *Metrics) RecordFork(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ForkDuration = append(m.ForkDuration, float64(d.Milliseconds()))
}

// RecordHibernate records hibernate latency and size
func (m *Metrics) RecordHibernate(d time.Duration, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HibernateDuration = append(m.HibernateDuration, float64(d.Milliseconds()))
	m.HibernateSizeBytes += size
}

// SumFloats returns the sum of a float64 slice
func SumFloats(a []float64) float64 {
	var s float64
	for _, v := range a {
		s += v
	}
	return s
}
