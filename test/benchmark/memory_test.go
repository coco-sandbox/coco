// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package benchmark

import (
	"fmt"
	"testing"
)

// MemoryOverheadTargetMiB is the spec target for per-sandbox control-plane
// memory overhead per spec §6.1 and §6.3.
const MemoryOverheadTargetMiB = 2.0

// ValidateMemoryOverhead checks if measured overhead meets spec target.
func ValidateMemoryOverhead(measuredMiB float64) error {
	if measuredMiB > MemoryOverheadTargetMiB {
		return fmt.Errorf("memory overhead %.2f MiB exceeds target %.2f MiB",
			measuredMiB, MemoryOverheadTargetMiB)
	}
	return nil
}

// TestMemoryOverheadTargetValidation verifies the benchmark framework.
func TestMemoryOverheadTargetValidation(t *testing.T) {
	valid := 1.5
	if err := ValidateMemoryOverhead(valid); err != nil {
		t.Errorf("expected valid results, got %v", err)
	}

	invalid := 3.0
	if err := ValidateMemoryOverhead(invalid); err == nil {
		t.Errorf("expected validation error for out-of-spec results")
	}
}

// BenchmarkMemoryOverhead is a placeholder that reports spec target.
func BenchmarkMemoryOverhead(b *testing.B) {
	b.ReportMetric(MemoryOverheadTargetMiB, "MiB_target")
	// Actual measurement requires: measure host RSS / sandbox count
}
