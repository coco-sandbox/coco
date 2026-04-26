package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	mu           sync.RWMutex
	lastCollect  time.Time
	collectCount uint64
	errors       uint64
}

func NewCollector() *Collector {
	return &Collector{
		lastCollect: time.Now(),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("coco_collector_last_collect",
		"Last collection timestamp",
		nil, nil,
	)
	ch <- prometheus.NewDesc("coco_collector_total",
		"Total number of collections",
		nil, nil,
	)
	ch <- prometheus.NewDesc("coco_collector_errors",
		"Total number of collection errors",
		nil, nil,
	)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("coco_collector_last_collect", "Last collection timestamp", nil, nil),
		prometheus.GaugeValue,
		float64(c.lastCollect.Unix()),
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("coco_collector_total", "Total number of collections", nil, nil),
		prometheus.GaugeValue,
		float64(c.collectCount),
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("coco_collector_errors", "Total number of collection errors", nil, nil),
		prometheus.GaugeValue,
		float64(c.errors),
	)

	c.lastCollect = time.Now()
	c.collectCount++
}

func (c *Collector) RecordError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors++
}

type MetricsCollector struct {
	mu         sync.RWMutex
	collectors []prometheus.Collector
	interval   time.Duration
	stopCh     chan struct{}
}

func NewMetricsCollector(interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		collectors: make([]prometheus.Collector, 0),
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

func (mc *MetricsCollector) AddCollector(c prometheus.Collector) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.collectors = append(mc.collectors, c)
}

func (mc *MetricsCollector) Start() {
	go mc.run()
}

func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
}

func (mc *MetricsCollector) run() {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopCh:
			return
		case <-ticker.C:
			mc.collect()
		}
	}
}

func (mc *MetricsCollector) collect() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, c := range mc.collectors {
		_ = c
	}
}

type SystemCollector struct {
	cpuUsage    float64
	memoryUsage float64
	diskUsage   float64
}

func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

func (sc *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("coco_system_cpu_usage", "CPU usage percentage", nil, nil)
	ch <- prometheus.NewDesc("coco_system_memory_usage", "Memory usage percentage", nil, nil)
	ch <- prometheus.NewDesc("coco_system_disk_usage", "Disk usage percentage", nil, nil)
}

func (sc *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("coco_system_cpu_usage", "CPU usage percentage", nil, nil),
		prometheus.GaugeValue,
		sc.cpuUsage,
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("coco_system_memory_usage", "Memory usage percentage", nil, nil),
		prometheus.GaugeValue,
		sc.memoryUsage,
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("coco_system_disk_usage", "Disk usage percentage", nil, nil),
		prometheus.GaugeValue,
		sc.diskUsage,
	)
}
