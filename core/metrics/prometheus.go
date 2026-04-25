// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MetricsCollector collects Prometheus-format metrics.
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters
	SandboxCreates   map[string]int // by template
	SandboxDestroys int
	ExecTotal       int

	// Gauges
	SandboxesByState map[string]int

	// Histograms (values for percentile calculation)
	CreateDurations  []float64
	ExecDurations   []float64
	ForkDurations   []float64
	HibernateDurations []float64
	HibernateSizes   []int64
	MemoryUsedBytes int64
	CPUSeconds      float64
	NetIngressBytes int64
	NetEgressBytes  int64
}

// New creates a new metrics collector.
func New() *MetricsCollector {
	return &MetricsCollector{
		SandboxCreates:   make(map[string]int),
		SandboxesByState: make(map[string]int),
		CreateDurations:  make([]float64, 0, 1000),
		ExecDurations:   make([]float64, 0, 10000),
		ForkDurations:   make([]float64, 0, 1000),
		HibernateDurations: make([]float64, 0, 100),
		HibernateSizes:  make([]int64, 0, 100),
	}
}

// RecordCreate records a sandbox creation.
func (m *MetricsCollector) RecordCreate(template string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SandboxCreates[template]++
	m.CreateDurations = append(m.CreateDurations, d.Seconds())
}

// RecordDestroy records a sandbox destruction.
func (m *MetricsCollector) RecordDestroy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SandboxDestroys++
}

// RecordExec records an exec operation.
func (m *MetricsCollector) RecordExec(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecTotal++
	m.ExecDurations = append(m.ExecDurations, d.Seconds())
}

// RecordFork records a fork operation.
func (m *MetricsCollector) RecordFork(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ForkDurations = append(m.ForkDurations, d.Seconds())
}

// RecordHibernate records a hibernate operation.
func (m *MetricsCollector) RecordHibernate(d time.Duration, sizeBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HibernateDurations = append(m.HibernateDurations, d.Seconds())
	m.HibernateSizes = append(m.HibernateSizes, sizeBytes)
}

// SetSandboxState updates the sandbox state gauge.
func (m *MetricsCollector) SetSandboxState(state string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SandboxesByState[state] = count
}

// SetMemoryUsage sets the current memory usage.
func (m *MetricsCollector) SetMemoryUsage(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MemoryUsedBytes = bytes
}

// String returns Prometheus-format metrics output.
func (m *MetricsCollector) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	// Sandbox counts by state
	sb.WriteString("# HELP coco_sandboxes_total Total number of sandboxes by state\n")
	sb.WriteString("# TYPE coco_sandboxes_total gauge\n")
	states := make([]string, 0, len(m.SandboxCreates))
	for s := range m.SandboxesByState {
		states = append(states, s)
	}
	sort.Strings(states)
	for _, s := range states {
		sb.WriteString(fmt.Sprintf("coco_sandboxes_total{state=%q} %d\n", s, m.SandboxesByState[s]))
	}

	// Create totals by template
	sb.WriteString("# HELP coco_sandbox_creates_total Total sandbox creates by template\n")
	sb.WriteString("# TYPE coco_sandbox_creates_total counter\n")
	templates := make([]string, 0, len(m.SandboxCreates))
	for t := range m.SandboxCreates {
		templates = append(templates, t)
	}
	sort.Strings(templates)
	for _, t := range templates {
		sb.WriteString(fmt.Sprintf("coco_sandbox_creates_total{template=%q} %d\n", t, m.SandboxCreates[t]))
	}

	// Destroy total
	sb.WriteString("# HELP coco_sandbox_destroys_total Total sandbox destroys\n")
	sb.WriteString("# TYPE coco_sandbox_destroys_total counter\n")
	sb.WriteString(fmt.Sprintf("coco_sandbox_destroys_total %d\n", m.SandboxDestroys))

	// Create duration histogram
	sb.WriteString("# HELP coco_sandbox_create_duration_seconds Sandbox create duration in seconds\n")
	sb.WriteString("# TYPE coco_sandbox_create_duration_seconds histogram\n")
	sb.WriteString(histogram("coco_sandbox_create_duration_seconds", m.CreateDurations))

	// Exec duration histogram
	sb.WriteString("# HELP coco_exec_duration_seconds Exec duration in seconds\n")
	sb.WriteString("# TYPE coco_exec_duration_seconds histogram\n")
	sb.WriteString(histogram("coco_exec_duration_seconds", m.ExecDurations))

	// Fork duration histogram
	sb.WriteString("# HELP coco_fork_duration_seconds Fork duration in seconds\n")
	sb.WriteString("# TYPE coco_fork_duration_seconds histogram\n")
	sb.WriteString(histogram("coco_fork_duration_seconds", m.ForkDurations))

	// Hibernate duration histogram
	sb.WriteString("# HELP coco_hibernate_duration_seconds Hibernate duration in seconds\n")
	sb.WriteString("# TYPE coco_hibernate_duration_seconds histogram\n")
	sb.WriteString(histogram("coco_hibernate_duration_seconds", m.HibernateDurations))

	// Hibernate size
	sb.WriteString("# HELP coco_hibernate_size_bytes Size of hibernate images in bytes\n")
	sb.WriteString("# TYPE coco_hibernate_size_bytes gauge\n")
	sb.WriteString(fmt.Sprintf("coco_hibernate_size_bytes %d\n", m.HibernateSizesAvg()))

	// Memory usage
	sb.WriteString("# HELP coco_memory_used_bytes Total memory used by sandboxes\n")
	sb.WriteString("# TYPE coco_memory_used_bytes gauge\n")
	sb.WriteString(fmt.Sprintf("coco_memory_used_bytes %d\n", m.MemoryUsedBytes))

	// CPU
	sb.WriteString("# HELP coco_cpu_seconds_total Total CPU time used\n")
	sb.WriteString("# TYPE coco_cpu_seconds_total counter\n")
	sb.WriteString(fmt.Sprintf("coco_cpu_seconds_total %.3f\n", m.CPUSeconds))

	// Network
	sb.WriteString("# HELP coco_network_bytes_total Total network bytes\n")
	sb.WriteString("# TYPE coco_network_bytes_total counter\n")
	sb.WriteString(fmt.Sprintf("coco_network_bytes_total{direction=%q} %d\n", "ingress", m.NetIngressBytes))
	sb.WriteString(fmt.Sprintf("coco_network_bytes_total{direction=%q} %d\n", "egress", m.NetEgressBytes))

	return sb.String()
}

func histogram(name string, values []float64) string {
	if len(values) == 0 {
		return fmt.Sprintf("%s_bucket{le=\"+Inf\"} 0\n", name)
	}

	var sb strings.Builder
	buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	sort.Float64s(values)

	for _, le := range buckets {
		count := 0
		for _, v := range values {
			if v <= le {
				count++
			}
		}
		sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%.3f\"} %d\n", name, le, count))
	}

	count := len(values)
	var sum float64
	for _, v := range values {
		sum += v
	}
	sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", name, count))
	sb.WriteString(fmt.Sprintf("%s_sum %.6f\n", name, sum))
	sb.WriteString(fmt.Sprintf("%s_count %d\n", name, count))

	return sb.String()
}

func (m *MetricsCollector) HibernateSizesAvg() int64 {
	if len(m.HibernateSizes) == 0 {
		return 0
	}
	var sum int64
	for _, s := range m.HibernateSizes {
		sum += s
	}
	return sum / int64(len(m.HibernateSizes))
}