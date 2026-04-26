// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package benchmark

import (
	"testing"
	"time"
)

// MeasureColdStart measures sandbox cold start latency.
// Per spec §6.1, cold start target is P50<30ms, P95<40ms, P99<50ms.
func MeasureColdStart(measurements []time.Duration) PercentileLatency {
	if len(measurements) == 0 {
		return PercentileLatency{}
	}

	copy := append([]time.Duration(nil), measurements...)
	sorted := sortDurations(copy)

	n := len(sorted)
	return PercentileLatency{
		P50: sorted[n*50/100],
		P95: sorted[n*95/100],
		P99: sorted[n*99/100],
	}
}

func sortDurations(d []time.Duration) []time.Duration {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
	return d
}

// TestColdStartTargetValidation verifies the benchmark framework works.
// Actual cold start measurement requires a running Coco cluster.
func TestColdStartTargetValidation(t *testing.T) {
	results := PercentileLatency{
		P50: 25 * time.Millisecond,
		P95: 38 * time.Millisecond,
		P99: 48 * time.Millisecond,
	}

	if err := ValidateColdStart(results); err != nil {
		t.Errorf("expected valid results, got %v", err)
	}

	badResults := PercentileLatency{
		P50: 35 * time.Millisecond,
		P95: 45 * time.Millisecond,
		P99: 55 * time.Millisecond,
	}

	if err := ValidateColdStart(badResults); err == nil {
		t.Errorf("expected validation error for out-of-spec results")
	}
}

// BenchmarkColdStart is a placeholder that reports spec targets.
// Real benchmark requires coco-visor running and sandbox creation endpoint.
func BenchmarkColdStart(b *testing.B) {
	target := ColdStartTarget()
	b.Logf("Spec targets: P50<%v, P95<%v, P99<%v", target.P50, target.P95, target.P99)
	b.ReportMetric(float64(target.P50.Milliseconds()), "ms/P50_target")
	b.ReportMetric(float64(target.P95.Milliseconds()), "ms/P95_target")
	b.ReportMetric(float64(target.P99.Milliseconds()), "ms/P99_target")
	// Actual measurement requires: create sandbox → measure time → destroy
	// For now, report targets so CI at least compiles benchmark tests
}