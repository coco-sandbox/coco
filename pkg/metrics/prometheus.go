// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	SandboxCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "coco_sandbox_count",
		Help: "Number of sandboxes in each state on each node (spec/08 §3.2)",
	}, []string{"state", "node"})

	SandboxCreateDurationMs = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "coco_sandbox_create_duration_ms",
		Help:    "Sandbox creation duration in milliseconds (spec/08 §3.2)",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 5000},
	}, []string{"template", "node"})

	SandboxForkDurationMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_sandbox_fork_duration_ms",
		Help:    "Sandbox fork duration in milliseconds (spec/08 §3.2)",
		Buckets: []float64{1, 2, 5, 10, 20, 50, 100, 500},
	})

	SandboxExecDurationMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_sandbox_exec_duration_ms",
		Help:    "Command execution duration in milliseconds (spec/08 §3.2)",
		Buckets: []float64{1, 5, 10, 50, 100, 500, 1000, 5000, 30000},
	})

	NodeMemoryUsedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "coco_node_memory_used_bytes",
		Help: "Memory used on each node in bytes (spec/08 §3.2)",
	}, []string{"node"})

	NodeCPUPercent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "coco_node_cpu_percent",
		Help: "CPU usage percentage on each node (spec/08 §3.2)",
	}, []string{"node"})

	NodeSandboxCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "coco_node_sandbox_count",
		Help: "Number of sandboxes on each node (spec/08 §3.2)",
	}, []string{"node"})

	PoolAvailable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_pool_available",
		Help: "Number of available VMs in the pool (spec/08 §3.2)",
	})

	PoolInUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_pool_in_use",
		Help: "Number of VMs in use (spec/08 §3.2)",
	})

	NetworkPacketsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_network_packets_total",
		Help: "Total packets processed (spec/08 §3.2)",
	})

	NetworkBytesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_network_bytes_total",
		Help: "Total bytes processed (spec/08 §3.2)",
	})
)

// Register registers all spec-mandated metrics with Prometheus.
func Register() {
	prometheus.MustRegister(
		SandboxCount,
		SandboxCreateDurationMs,
		SandboxForkDurationMs,
		SandboxExecDurationMs,
		NodeMemoryUsedBytes,
		NodeCPUPercent,
		NodeSandboxCount,
		PoolAvailable,
		PoolInUse,
		NetworkPacketsTotal,
		NetworkBytesTotal,
	)
}

// Handler returns the Prometheus metrics handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordSandboxCreate observes a sandbox creation latency in milliseconds.
func RecordSandboxCreate(template, node string, durationMs float64) {
	SandboxCreateDurationMs.WithLabelValues(template, node).Observe(durationMs)
}

// RecordSandboxFork observes a sandbox fork latency in milliseconds.
func RecordSandboxFork(durationMs float64) {
	SandboxForkDurationMs.Observe(durationMs)
}

// RecordSandboxExec observes a command execution latency in milliseconds.
func RecordSandboxExec(durationMs float64) {
	SandboxExecDurationMs.Observe(durationMs)
}

// SetSandboxCount sets the gauge for the count of sandboxes in a given state on a node.
func SetSandboxCount(state, node string, n float64) {
	SandboxCount.WithLabelValues(state, node).Set(n)
}
