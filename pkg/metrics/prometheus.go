package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SandboxCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_sandbox_count",
		Help: "Number of active sandboxes",
	})

	SandboxCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "coco_sandbox_created_total",
		Help: "Total number of sandboxes created",
	})

	SandboxDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "coco_sandbox_deleted_total",
		Help: "Total number of sandboxes deleted",
	})

	SandboxDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_sandbox_duration_seconds",
		Help:    "Sandbox lifetime in seconds",
		Buckets: prometheus.DefBuckets,
	})

	ExecTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coco_exec_total",
		Help: "Total number of executions",
	}, []string{"sandbox_id", "exit_code"})

	ExecDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_exec_duration_seconds",
		Help:    "Execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	NodeCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_count",
		Help: "Number of active nodes",
	})

	NodeHealthyCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_healthy_count",
		Help: "Number of healthy nodes",
	})

	VMCreateDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_vm_create_duration_seconds",
		Help:    "VM creation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	ForkDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_fork_duration_seconds",
		Help:    "Fork operation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	CheckpointDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_checkpoint_duration_seconds",
		Help:    "Checkpoint duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	MemoryUsageBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "coco_memory_usage_bytes",
		Help: "Memory usage by sandbox",
	}, []string{"sandbox_id"})

	CPUUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "coco_cpu_usage",
		Help: "CPU usage by sandbox",
	}, []string{"sandbox_id"})

	NetworkBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coco_network_bytes_total",
		Help: "Total network bytes transferred",
	}, []string{"sandbox_id", "direction"})
)

func RecordSandboxCreated() {
	SandboxCreatedTotal.Inc()
	SandboxCount.Inc()
}

func RecordSandboxDeleted() {
	SandboxDeletedTotal.Inc()
	SandboxCount.Dec()
}

func RecordExec(sandboxID, exitCode string, duration float64) {
	ExecTotal.WithLabelValues(sandboxID, exitCode).Inc()
	ExecDuration.Observe(duration)
}

func RecordVMCreate(duration float64) {
	VMCreateDuration.Observe(duration)
}

func RecordFork(duration float64) {
	ForkDuration.Observe(duration)
}

func RecordCheckpoint(duration float64) {
	CheckpointDuration.Observe(duration)
}
