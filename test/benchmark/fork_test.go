// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package benchmark

import (
	"sort"
	"testing"
	"time"
)

// MeasureFork measures sandbox fork latency.
// Per spec §6.1, fork target is P50<10ms, P95<15ms, P99<20ms.
func MeasureFork(measurements []time.Duration) PercentileLatency {
	if len(measurements) == 0 {
		return PercentileLatency{}
	}

	sorted := append([]time.Duration(nil), measurements...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	return PercentileLatency{
		P50: sorted[n*50/100],
		P95: sorted[n*95/100],
		P99: sorted[n*99/100],
	}
}

// TestMeasureFork verifies the measurement logic.
func TestMeasureFork(t *testing.T) {
	measurements := make([]time.Duration, 100)
	for i := range measurements {
		measurements[i] = time.Duration(5+i%15) * time.Millisecond
	}

	result := MeasureFork(measurements)
	if result.P50 == 0 || result.P95 == 0 || result.P99 == 0 {
		t.Errorf("expected non-zero percentiles, got %+v", result)
	}
}

// BenchmarkFork is a placeholder that reports spec targets.
func BenchmarkFork(b *testing.B) {
	target := ForkTarget()
	b.Logf("Spec targets: P50<%v, P95<%v, P99<%v", target.P50, target.P95, target.P99)
	b.ReportMetric(float64(target.P50.Milliseconds()), "ms/P50_target")
	b.ReportMetric(float64(target.P95.Milliseconds()), "ms/P95_target")
	b.ReportMetric(float64(target.P99.Milliseconds()), "ms/P99_target")
}
