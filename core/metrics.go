// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
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
	checks := map[string]bool{
		"visor_socket": false,
		"net_socket":   false,
		"badger_db":    false,
		"kvm":          false,
	}

	// Check visor socket
	_, err := os.Stat("/run/coco/visor.sock")
	checks["visor_socket"] = err == nil

	// Check net socket
	_, err = os.Stat("/run/coco/net.sock")
	checks["net_socket"] = err == nil

	// Check BadgerDB directory
	_, err = os.Stat("/var/lib/coco/store")
	checks["badger_db"] = err == nil

	// Check KVM availability
	_, err = os.Stat("/dev/kvm")
	checks["kvm"] = err == nil

	allReady := true
	for _, ready := range checks {
		if !ready {
			allReady = false
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ready":  allReady,
		"checks": checks,
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
	// Bucket boundaries: 5, 10, 20, 30, 50, 100, 200, 500, 1000, +Inf (ms)
	buckets := []float64{5, 10, 20, 30, 50, 100, 200, 500, 1000}
	bucketCounts := make([]int, len(buckets)+1) // +1 for +Inf bucket
	for _, d := range state.metrics.createDuration {
		found := false
		for i, bound := range buckets {
			if d <= bound {
				bucketCounts[i]++
				found = true
				break
			}
		}
		if !found {
			bucketCounts[len(buckets)]++ // +Inf bucket
		}
	}
	cumulative := 0
	for i, bound := range buckets {
		cumulative += bucketCounts[i]
		lines = append(lines, fmt.Sprintf("coco_boot_duration_ms_bucket{le=\"%.0f\"} %d", bound, cumulative))
	}
	cumulative += bucketCounts[len(buckets)]
	lines = append(lines, fmt.Sprintf("coco_boot_duration_ms_bucket{le=\"+Inf\"} %d", cumulative))
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
	lines = append(lines, "# HELP coco_fork_duration_ms Fork latency in milliseconds")
	lines = append(lines, "# TYPE coco_fork_duration_ms histogram")
	forkBuckets := []float64{5, 10, 15, 20, 30, 50, 100}
	forkCounts := make([]int, len(forkBuckets)+1)
	for _, d := range state.metrics.forkDuration {
		found := false
		for i, bound := range forkBuckets {
			if d <= bound {
				forkCounts[i]++
				found = true
				break
			}
		}
		if !found {
			forkCounts[len(forkBuckets)]++
		}
	}
	cumulative = 0
	for i, bound := range forkBuckets {
		cumulative += forkCounts[i]
		lines = append(lines, fmt.Sprintf("coco_fork_duration_ms_bucket{le=\"%.0f\"} %d", bound, cumulative))
	}
	cumulative += forkCounts[len(forkBuckets)]
	lines = append(lines, fmt.Sprintf("coco_fork_duration_ms_bucket{le=\"+Inf\"} %d", cumulative))
	lines = append(lines, fmt.Sprintf("coco_fork_duration_ms_sum %.2f", sumFloats(state.metrics.forkDuration)))
	lines = append(lines, fmt.Sprintf("coco_fork_duration_ms_count %d", len(state.metrics.forkDuration)))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_hibernate_duration_ms Hibernate latency in milliseconds")
	lines = append(lines, "# TYPE coco_hibernate_duration_ms histogram")
	hibBuckets := []float64{500, 1000, 1500, 2000, 3000, 5000}
	hibCounts := make([]int, len(hibBuckets)+1)
	for _, d := range state.metrics.hibernateDuration {
		found := false
		for i, bound := range hibBuckets {
			if d <= bound {
				hibCounts[i]++
				found = true
				break
			}
		}
		if !found {
			hibCounts[len(hibBuckets)]++
		}
	}
	cumulative = 0
	for i, bound := range hibBuckets {
		cumulative += hibCounts[i]
		lines = append(lines, fmt.Sprintf("coco_hibernate_duration_ms_bucket{le=\"%.0f\"} %d", bound, cumulative))
	}
	cumulative += hibCounts[len(hibBuckets)]
	lines = append(lines, fmt.Sprintf("coco_hibernate_duration_ms_bucket{le=\"+Inf\"} %d", cumulative))
	lines = append(lines, fmt.Sprintf("coco_hibernate_duration_ms_sum %.2f", sumFloats(state.metrics.hibernateDuration)))
	lines = append(lines, fmt.Sprintf("coco_hibernate_duration_ms_count %d", len(state.metrics.hibernateDuration)))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_hibernate_bytes_total Bytes written during hibernate")
	lines = append(lines, "# TYPE coco_hibernate_bytes_total counter")
	lines = append(lines, fmt.Sprintf("coco_hibernate_bytes_total %d", state.metrics.hibernateSizeBytes))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_exec_duration_ms Exec latency in milliseconds")
	lines = append(lines, "# TYPE coco_exec_duration_ms histogram")
	execBuckets := []float64{1, 5, 10, 50, 100, 500, 1000, 5000}
	execCounts := make([]int, len(execBuckets)+1)
	for _, d := range state.metrics.execDuration {
		found := false
		for i, bound := range execBuckets {
			if d <= bound {
				execCounts[i]++
				found = true
				break
			}
		}
		if !found {
			execCounts[len(execBuckets)]++
		}
	}
	cumulative = 0
	for i, bound := range execBuckets {
		cumulative += execCounts[i]
		lines = append(lines, fmt.Sprintf("coco_exec_duration_ms_bucket{le=\"%.0f\"} %d", bound, cumulative))
	}
	cumulative += execCounts[len(execBuckets)]
	lines = append(lines, fmt.Sprintf("coco_exec_duration_ms_bucket{le=\"+Inf\"} %d", cumulative))
	lines = append(lines, fmt.Sprintf("coco_exec_duration_ms_sum %.2f", sumFloats(state.metrics.execDuration)))
	lines = append(lines, fmt.Sprintf("coco_exec_duration_ms_count %d", len(state.metrics.execDuration)))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_network_bytes_total Network bytes")
	lines = append(lines, "# TYPE coco_network_bytes_total counter")
	lines = append(lines, fmt.Sprintf("coco_network_bytes_total{direction=\"ingress\"} %d", state.metrics.networkBytesIngress))
	lines = append(lines, fmt.Sprintf("coco_network_bytes_total{direction=\"egress\"} %d", state.metrics.networkBytesEgress))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_memory_bytes_total Memory bytes used by sandboxes")
	lines = append(lines, "# TYPE coco_memory_bytes_total gauge")
	lines = append(lines, fmt.Sprintf("coco_memory_bytes_total %d", state.metrics.memoryUsedBytes))

	lines = append(lines, "")
	lines = append(lines, "# HELP coco_cpu_seconds_total CPU time used")
	lines = append(lines, "# TYPE coco_cpu_seconds_total counter")
	lines = append(lines, fmt.Sprintf("coco_cpu_seconds_total %.2f", state.metrics.cpuSecondsTotal))

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