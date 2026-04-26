// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Metrics collection for coco-visor.
//! Spec: Prometheus metrics on port 9090, histograms for durations, gauges for state.

const std = @import("std");

pub const Metrics = struct {
    boot_count: u64 = 0,
    fork_count: u64 = 0,
    checkpoint_count: u64 = 0,
    hibernate_count: u64 = 0,
    resume_count: u64 = 0,
    pause_count: u64 = 0,
    destroy_count: u64 = 0,
    exec_count: u64 = 0,
    total_boot_time_ns: u64 = 0,
    total_fork_time_ns: u64 = 0,
    total_checkpoint_time_ns: u64 = 0,
    total_hibernate_time_ns: u64 = 0,
    total_exec_time_ns: u64 = 0,
    active_sandboxes: u64 = 0,
    paused_sandboxes: u64 = 0,
    hibernated_sandboxes: u64 = 0,

    boot_times: [64]u64 = undefined,
    boot_times_idx: usize = 0,
    fork_times: [64]u64 = undefined,
    fork_times_idx: usize = 0,
    checkpoint_times: [64]u64 = undefined,
    checkpoint_times_idx: usize = 0,

    pub fn recordBoot(self: *Metrics, duration_ns: u64) void {
        self.boot_count += 1;
        self.total_boot_time_ns += duration_ns;
        self.boot_times[self.boot_times_idx % 64] = duration_ns;
        self.boot_times_idx += 1;
    }

    pub fn recordFork(self: *Metrics, duration_ns: u64) void {
        self.fork_count += 1;
        self.total_fork_time_ns += duration_ns;
        self.fork_times[self.fork_times_idx % 64] = duration_ns;
        self.fork_times_idx += 1;
    }

    pub fn recordCheckpoint(self: *Metrics, duration_ns: u64) void {
        self.checkpoint_count += 1;
        self.total_checkpoint_time_ns += duration_ns;
        self.checkpoint_times[self.checkpoint_times_idx % 64] = duration_ns;
        self.checkpoint_times_idx += 1;
    }

    pub fn recordHibernate(self: *Metrics, duration_ns: u64) void {
        self.hibernate_count += 1;
        self.total_hibernate_time_ns += duration_ns;
    }

    pub fn recordResume(self: *Metrics) void {
        self.resume_count += 1;
    }

    pub fn recordPause(self: *Metrics) void {
        self.pause_count += 1;
    }

    pub fn recordDestroy(self: *Metrics) void {
        self.destroy_count += 1;
    }

    pub fn recordExec(self: *Metrics, duration_ns: u64) void {
        self.exec_count += 1;
        self.total_exec_time_ns += duration_ns;
    }

    pub fn incActiveSandboxes(self: *Metrics) void {
        self.active_sandboxes += 1;
    }

    pub fn decActiveSandboxes(self: *Metrics) void {
        if (self.active_sandboxes > 0) self.active_sandboxes -= 1;
    }

    pub fn incPausedSandboxes(self: *Metrics) void {
        self.paused_sandboxes += 1;
    }

    pub fn decPausedSandboxes(self: *Metrics) void {
        if (self.paused_sandboxes > 0) self.paused_sandboxes -= 1;
    }

    pub fn incHibernatedSandboxes(self: *Metrics) void {
        self.hibernated_sandboxes += 1;
    }

    pub fn decHibernatedSandboxes(self: *Metrics) void {
        if (self.hibernated_sandboxes > 0) self.hibernated_sandboxes -= 1;
    }

    pub fn avgBootTimeNs(self: *Metrics) u64 {
        if (self.boot_count == 0) return 0;
        return self.total_boot_time_ns / self.boot_count;
    }

    pub fn avgForkTimeNs(self: *Metrics) u64 {
        if (self.fork_count == 0) return 0;
        return self.total_fork_time_ns / self.fork_count;
    }

    pub fn p50BootTimeNs(self: *Metrics) u64 {
        if (self.boot_times_idx == 0) return 0;
        return self.boot_times[self.boot_times_idx / 2];
    }

    pub fn p99BootTimeNs(self: *Metrics) u64 {
        if (self.boot_times_idx == 0) return 0;
        const idx = (self.boot_times_idx * 99) / 100;
        return self.boot_times[idx];
    }

    pub fn formatPrometheus(self: *Metrics, writer: std.io.AnyWriter) !void {
        try writer.writeAll("# HELP coco_sandbox_count Current number of sandboxes by state.\n");
        try writer.writeAll("# TYPE coco_sandbox_count gauge\n");
        try writer.print("coco_sandbox_count{{state=\"active\"}} {d}\n", .{self.active_sandboxes});
        try writer.print("coco_sandbox_count{{state=\"paused\"}} {d}\n", .{self.paused_sandboxes});
        try writer.print("coco_sandbox_count{{state=\"hibernated\"}} {d}\n", .{self.hibernated_sandboxes});

        try writer.writeAll("\n# HELP coco_sandbox_operations_total Total number of sandbox operations.\n");
        try writer.writeAll("# TYPE coco_sandbox_operations_total counter\n");
        try writer.print("coco_sandbox_operations_total{{operation=\"boot\"}} {d}\n", .{self.boot_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"fork\"}} {d}\n", .{self.fork_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"checkpoint\"}} {d}\n", .{self.checkpoint_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"hibernate\"}} {d}\n", .{self.hibernate_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"resume\"}} {d}\n", .{self.resume_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"pause\"}} {d}\n", .{self.pause_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"destroy\"}} {d}\n", .{self.destroy_count});
        try writer.print("coco_sandbox_operations_total{{operation=\"exec\"}} {d}\n", .{self.exec_count});

        try writer.writeAll("\n# HELP coco_sandbox_operation_duration_ms Duration of sandbox operations in milliseconds.\n");
        try writer.writeAll("# TYPE coco_sandbox_operation_duration_ms histogram\n");
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"boot\",le=\"1\"}} {d}\n", .{if (self.p50BootTimeNs() < 1_000_000) self.boot_count else 0});
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"boot\",le=\"10\"}} {d}\n", .{if (self.p50BootTimeNs() < 10_000_000) self.boot_count else 0});
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"boot\",le=\"30\"}} {d}\n", .{if (self.p50BootTimeNs() < 30_000_000) self.boot_count else 0});
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"boot\",le=\"100\"}} {d}\n", .{if (self.p50BootTimeNs() < 100_000_000) self.boot_count else 0});
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"boot\",le=\"+Inf\"}} {d}\n", .{self.boot_count});
        try writer.print("coco_sandbox_operation_duration_ms_sum{{operation=\"boot\"}} {d}\n", .{self.total_boot_time_ns / 1_000_000});
        try writer.print("coco_sandbox_operation_duration_ms_count{{operation=\"boot\"}} {d}\n", .{self.boot_count});

        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"fork\",le=\"1\"}} {d}\n", .{if (self.avgForkTimeNs() < 1_000_000) self.fork_count else 0});
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"fork\",le=\"10\"}} {d}\n", .{if (self.avgForkTimeNs() < 10_000_000) self.fork_count else 0});
        try writer.print("coco_sandbox_operation_duration_ms_bucket{{operation=\"fork\",le=\"+Inf\"}} {d}\n", .{self.fork_count});
        try writer.print("coco_sandbox_operation_duration_ms_sum{{operation=\"fork\"}} {d}\n", .{self.total_fork_time_ns / 1_000_000});
        try writer.print("coco_sandbox_operation_duration_ms_count{{operation=\"fork\"}} {d}\n", .{self.fork_count});
    }
};

var global_metrics: Metrics = .{};

pub fn getMetrics() *Metrics {
    return &global_metrics;
}
