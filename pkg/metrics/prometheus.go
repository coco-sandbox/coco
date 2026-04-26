// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	SandboxCreatedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_sandbox_created_total",
		Help: "Total number of sandboxes created",
	})

	SandboxRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_sandbox_running",
		Help: "Number of currently running sandboxes",
	})

	SandboxPaused = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_sandbox_paused",
		Help: "Number of currently paused sandboxes",
	})

	SandboxHibernated = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_sandbox_hibernated",
		Help: "Number of currently hibernated sandboxes",
	})

	ExecDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "coco_exec_duration_seconds",
		Help:    "Execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"sandbox_id", "exit_code"})

	BootDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_boot_duration_seconds",
		Help:    "VM boot duration in seconds",
		Buckets: []float64{.01, .05, .1, .5, 1, 5},
	})

	ForkDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_fork_duration_seconds",
		Help:    "Sandbox fork duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	HibernateDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_hibernate_duration_seconds",
		Help:    "Sandbox hibernate duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	NetworkSessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_network_sessions_active",
		Help: "Number of active NAT sessions",
	})

	ReplayEventsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_replay_events_total",
		Help: "Total number of replay events recorded",
	})

	CheckpointCreatedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_checkpoint_created_total",
		Help: "Total number of checkpoints created",
	})

	NodeMemoryUsedBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_memory_used_bytes",
		Help: "Memory used on each node in bytes",
	})

	NodeCPUPercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_cpu_percent",
		Help: "CPU usage percentage on each node",
	})

	NodeSandboxCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_sandbox_count",
		Help: "Number of sandboxes on each node",
	})

	PoolAvailable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_pool_available",
		Help: "Number of available VMs in the pool",
	})

	PoolInUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coco_pool_in_use",
		Help: "Number of VMs in use",
	})

	NetworkPacketsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_network_packets_total",
		Help: "Total packets processed",
	})

	NetworkBytesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "coco_network_bytes_total",
		Help: "Total bytes processed",
	})
)

// Register registers all metrics with Prometheus
func Register() {
	prometheus.MustRegister(
		SandboxCreatedTotal,
		SandboxRunning,
		SandboxPaused,
		SandboxHibernated,
		ExecDuration,
		BootDuration,
		ForkDuration,
		HibernateDuration,
		NetworkSessionsActive,
		ReplayEventsTotal,
		CheckpointCreatedTotal,
		NodeMemoryUsedBytes,
		NodeCPUPercent,
		NodeSandboxCount,
		PoolAvailable,
		PoolInUse,
		NetworkPacketsTotal,
		NetworkBytesTotal,
	)
}

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordBoot records a VM boot completion
func RecordBoot(durationSeconds float64) {
	BootDuration.Observe(durationSeconds)
	SandboxCreatedTotal.Inc()
	SandboxRunning.Inc()
}

// RecordExec records an exec completion
func RecordExec(sandboxID string, exitCode int, durationSeconds float64) {
	ExecDuration.WithLabelValues(sandboxID, string(rune('0'+exitCode))).Observe(durationSeconds)
}

// RecordFork records a fork completion
func RecordFork(durationSeconds float64) {
	ForkDuration.Observe(durationSeconds)
}

// RecordHibernate records a hibernate completion
func RecordHibernate(durationSeconds float64) {
	HibernateDuration.Observe(durationSeconds)
	SandboxRunning.Dec()
	SandboxHibernated.Inc()
}

// RecordResume records a resume from hibernate
func RecordResume() {
	SandboxHibernated.Dec()
	SandboxRunning.Inc()
}

// RecordCheckpoint records checkpoint creation
func RecordCheckpoint() {
	CheckpointCreatedTotal.Inc()
}

// RecordReplayEvent records a replay event
func RecordReplayEvent() {
	ReplayEventsTotal.Inc()
}
