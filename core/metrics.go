// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Health / Ready
// =============================================================================

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"healthy": true,
		"version": "0.1.0",
	})
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	_, err := os.Stat("/run/coco/visor.sock")
	ready := err == nil
	writeJSON(w, http.StatusOK, map[string]any{
		"ready":       ready,
		"visor_socket": ready,
	})
}

// =============================================================================
// Metrics
// =============================================================================

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	state.metrics.mu.RLock()
	defer state.metrics.mu.RUnlock()

	var lines []string
	lines = append(lines, "# HELP coco_sandbox_count Number of sandboxes by state")
	lines = append(lines, "# TYPE coco_sandbox_count gauge")
	for state, count := range state.metrics.sandboxesTotal {
		lines = append(lines, fmt.Sprintf("coco_sandbox_count{state=%q} %d", state, count))
	}

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_boot_duration_ms Boot latency in milliseconds")
	lines = append(lines, "# TYPE coco_boot_duration_ms histogram")
	for i, d := range state.metrics.createDuration {
		lines = append(lines, fmt.Sprintf("coco_boot_duration_ms_bucket{le=%d} %d", (i+1)*10, int(d)))
	}
	lines = append(lines, fmt.Sprintf("coco_boot_duration_ms_sum %.2f", sumFloats(state.metrics.createDuration)))
	lines = append(lines, fmt.Sprintf("coco_boot_duration_ms_count %d", len(state.metrics.createDuration)))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_creates_total Total sandbox creates by template")
	lines = append(lines, "# TYPE coco_creates_total counter")
	for tmpl, count := range state.metrics.createsTotal {
		lines = append(lines, fmt.Sprintf("coco_creates_total{template=%q} %d", tmpl, count))
	}

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_destroys_total Total sandbox destroys")
	lines = append(lines, "# TYPE coco_destroys_total counter")
	lines = append(lines, fmt.Sprintf("coco_destroys_total %d", state.metrics.destroysTotal))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_network_bytes_total Network bytes")
	lines = append(lines, "# TYPE coco_network_bytes_total counter")
	lines = append(lines, fmt.Sprintf("coco_network_bytes_total{direction=\"ingress\"} %d", state.metrics.networkBytesIngress))
	lines = append(lines, fmt.Sprintf("coco_network_bytes_total{direction=\"egress\"} %d", state.metrics.networkBytesEgress))

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write([]byte(strings.Join(lines, "\n")))
}

// =============================================================================
// Metrics Tracking
// =============================================================================

func newMetrics() *Metrics {
	return &Metrics{
		sandboxesTotal: make(map[string]int),
		createsTotal:   make(map[string]int),
		createDuration: make([]float64, 0, 100),
		execDuration:   make([]float64, 0, 100),
		forkDuration:   make([]float64, 0, 100),
		hibernateDuration: make([]float64, 0, 100),
	}
}

func (m *Metrics) RecordCreate(template string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createsTotal[template]++
	m.createDuration = append(m.createDuration, float64(d.Milliseconds()))
	m.sandboxesTotal["creating"]++
	m.sandboxesTotal["running"]++
}

func (m *Metrics) RecordDestroy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.destroysTotal++
	if c, ok := m.sandboxesTotal["running"]; ok && c > 0 {
		m.sandboxesTotal["running"]--
	}
	m.sandboxesTotal["destroyed"]++
}

func (m *Metrics) RecordExec(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execDuration = append(m.execDuration, float64(d.Milliseconds()))
}

func (m *Metrics) RecordFork(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forkDuration = append(m.forkDuration, float64(d.Milliseconds()))
}

func (m *Metrics) RecordHibernate(d time.Duration, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hibernateDuration = append(m.hibernateDuration, float64(d.Milliseconds()))
	m.hibernateSizeBytes += size
}

func sumFloats(a []float64) float64 {
	var s float64
	for _, v := range a {
		s += v
	}
	return s
}

func quantileFloat64(a []float64, q float64) float64 {
	if len(a) == 0 {
		return 0
	}
	sorted := make([]float64, len(a))
	copy(sorted, a)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1)*q)
	return sorted[idx]
}