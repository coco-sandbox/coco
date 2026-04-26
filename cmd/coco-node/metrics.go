package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	sandboxCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_sandbox_count",
		Help: "Number of active sandboxes on this node",
	})

	sandboxCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "coco_node_sandbox_created_total",
		Help: "Total number of sandboxes created on this node",
	})

	sandboxDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "coco_node_sandbox_deleted_total",
		Help: "Total number of sandboxes deleted on this node",
	})

	vmPoolFree = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_vm_pool_free",
		Help: "Number of free VMs in the pool",
	})

	vmPoolActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_vm_pool_active",
		Help: "Number of active VMs in the pool",
	})

	nodeMemoryUsedMB = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_memory_used_mb",
		Help: "Memory used by sandboxes in MB",
	})

	nodeCPUUsed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coco_node_cpu_used",
		Help: "Number of vCPUs in use",
	})

	execTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coco_node_exec_total",
		Help: "Total number of executions",
	}, []string{"sandbox_id", "exit_code"})

	execDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coco_node_exec_duration_seconds",
		Help:    "Execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	})
)

type NodeMetrics struct {
	mu        sync.RWMutex
	nodeID    string
	startTime time.Time
	registry  *prometheus.Registry
	collector *metricsCollector
}

type metricsCollector struct {
	nodeID string
}

func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sandboxCount.Desc()
	ch <- vmPoolFree.Desc()
	ch <- vmPoolActive.Desc()
}

func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		sandboxCount.Desc(),
		prometheus.GaugeValue,
		0,
		c.nodeID,
	)
}

func NewNodeMetrics(nodeID string) *NodeMetrics {
	registry := prometheus.NewRegistry()
	collector := &metricsCollector{nodeID: nodeID}

	registry.MustRegister(sandboxCount)
	registry.MustRegister(sandboxCreatedTotal)
	registry.MustRegister(sandboxDeletedTotal)
	registry.MustRegister(vmPoolFree)
	registry.MustRegister(vmPoolActive)
	registry.MustRegister(nodeMemoryUsedMB)
	registry.MustRegister(nodeCPUUsed)
	registry.MustRegister(execTotal)
	registry.MustRegister(execDuration)

	return &NodeMetrics{
		nodeID:    nodeID,
		startTime: time.Now(),
		registry:  registry,
		collector: collector,
	}
}

func (nm *NodeMetrics) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nm.update()
		}
	}
}

func (nm *NodeMetrics) UpdateSandboxCount(count int) {
	sandboxCount.Set(float64(count))
}

func (nm *NodeMetrics) IncSandboxCreated() {
	sandboxCreatedTotal.Inc()
}

func (nm *NodeMetrics) IncSandboxDeleted() {
	sandboxDeletedTotal.Inc()
}

func (nm *NodeMetrics) UpdatePoolMetrics(free, active int) {
	vmPoolFree.Set(float64(free))
	vmPoolActive.Set(float64(active))
}

func (nm *NodeMetrics) UpdateResourceUsage(memUsedMB uint64, cpuUsed int) {
	nodeMemoryUsedMB.Set(float64(memUsedMB))
	nodeCPUUsed.Set(float64(cpuUsed))
}

func (nm *NodeMetrics) RecordExec(sandboxID, exitCode string, duration time.Duration) {
	execTotal.WithLabelValues(sandboxID, exitCode).Inc()
	execDuration.Observe(duration.Seconds())
}

func (nm *NodeMetrics) GetRegistry() *prometheus.Registry {
	return nm.registry
}

func (nm *NodeMetrics) update() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	log.Printf("Metrics updated at %s", time.Now().Format(time.RFC3339))
}

func (nm *NodeMetrics) Close() error {
	return nil
}
