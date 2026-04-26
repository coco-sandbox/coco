// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package benchmark

import (
	"fmt"
	"testing"
)

// Network targets per spec §6.1 and §7.
const (
	// NetworkThroughputTargetGbps is the aggregate throughput target per node.
	NetworkThroughputTargetGbps = 20.0
	// NetworkRTTTargetMs is the host-to-guest RTT target.
	NetworkRTTTargetMs = 0.5
)

// ValidateNetworkThroughput checks if measured throughput meets spec target.
func ValidateNetworkThroughput(measuredGbps float64) error {
	if measuredGbps < NetworkThroughputTargetGbps {
		return fmt.Errorf("network throughput %.2f Gbps below target %.2f Gbps",
			measuredGbps, NetworkThroughputTargetGbps)
	}
	return nil
}

// ValidateNetworkRTT checks if measured RTT meets spec target.
func ValidateNetworkRTT(measuredMs float64) error {
	if measuredMs > NetworkRTTTargetMs {
		return fmt.Errorf("network RTT %.2f ms exceeds target %.2f ms",
			measuredMs, NetworkRTTTargetMs)
	}
	return nil
}

// TestNetworkTargetValidation verifies the benchmark framework.
func TestNetworkTargetValidation(t *testing.T) {
	if err := ValidateNetworkThroughput(22.0); err != nil {
		t.Errorf("expected valid throughput, got %v", err)
	}
	if err := ValidateNetworkThroughput(15.0); err == nil {
		t.Errorf("expected error for throughput below target")
	}

	if err := ValidateNetworkRTT(0.3); err != nil {
		t.Errorf("expected valid RTT, got %v", err)
	}
	if err := ValidateNetworkRTT(0.8); err == nil {
		t.Errorf("expected error for RTT above target")
	}
}

// BenchmarkNetwork is a placeholder that reports spec targets.
func BenchmarkNetwork(b *testing.B) {
	b.ReportMetric(NetworkThroughputTargetGbps, "Gbps_target")
	b.ReportMetric(NetworkRTTTargetMs, "ms_rtt_target")
	// Actual measurement requires: send traffic through coco-net XDP → measure bandwidth/RTT
}
