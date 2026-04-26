// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Metrics collection for coco-visor.

const std = @import("std");

pub const Metrics = struct {
    boot_count: u64 = 0,
    fork_count: u64 = 0,
    checkpoint_count: u64 = 0,
    total_boot_time_ns: u64 = 0,
    total_fork_time_ns: u64 = 0,
    total_checkpoint_time_ns: u64 = 0,

    pub fn recordBoot(self: *Metrics, duration_ns: u64) void {
        self.boot_count += 1;
        self.total_boot_time_ns += duration_ns;
    }

    pub fn recordFork(self: *Metrics, duration_ns: u64) void {
        self.fork_count += 1;
        self.total_fork_time_ns += duration_ns;
    }

    pub fn recordCheckpoint(self: *Metrics, duration_ns: u64) void {
        self.checkpoint_count += 1;
        self.total_checkpoint_time_ns += duration_ns;
    }

    pub fn avgBootTimeNs(self: *Metrics) u64 {
        if (self.boot_count == 0) return 0;
        return self.total_boot_time_ns / self.boot_count;
    }
};

var global_metrics: Metrics = .{};

pub fn getMetrics() *Metrics {
    return &global_metrics;
}
