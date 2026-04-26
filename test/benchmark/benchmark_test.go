// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package benchmark

import (
	"fmt"
	"testing"
	"time"
)

// CocoBenchmarkResults holds benchmark results per spec §6.
type CocoBenchmarkResults struct {
	ColdStartMs  PercentileLatency
	ForkMs       PercentileLatency
	MemoryOverheadMiB float64
	NetworkGbps  float64
	NetworkRTTMs float64
}

// PercentileLatency holds P50/P95/P99 latency targets per spec §6 Table.
type PercentileLatency struct {
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
}

// ColdStartTarget returns the spec target for cold start per spec §6.1.
func ColdStartTarget() PercentileLatency {
	return PercentileLatency{
		P50: 30 * time.Millisecond,
		P95: 40 * time.Millisecond,
		P99: 50 * time.Millisecond,
	}
}

// ForkTarget returns the spec target for fork per spec §6.1.
func ForkTarget() PercentileLatency {
	return PercentileLatency{
		P50: 10 * time.Millisecond,
		P95: 15 * time.Millisecond,
		P99: 20 * time.Millisecond,
	}
}

// ValidateColdStart checks if results meet spec targets.
func ValidateColdStart(results PercentileLatency) error {
	target := ColdStartTarget()
	if results.P50 > target.P50 {
		return fmt.Errorf("cold start P50 %v exceeds target %v", results.P50, target.P50)
	}
	if results.P95 > target.P95 {
		return fmt.Errorf("cold start P95 %v exceeds target %v", results.P95, target.P95)
	}
	if results.P99 > target.P99 {
		return fmt.Errorf("cold start P99 %v exceeds target %v", results.P99, target.P99)
	}
	return nil
}

// ValidateFork checks if results meet spec targets.
func ValidateFork(results PercentileLatency) error {
	target := ForkTarget()
	if results.P50 > target.P50 {
		return fmt.Errorf("fork P50 %v exceeds target %v", results.P50, target.P50)
	}
	if results.P95 > target.P95 {
		return fmt.Errorf("fork P95 %v exceeds target %v", results.P95, target.P95)
	}
	if results.P99 > target.P99 {
		return fmt.Errorf("fork P99 %v exceeds target %v", results.P99, target.P99)
	}
	return nil
}