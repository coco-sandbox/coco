// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Snapshot operations for VM state capture.

const std = @import("std");
const posix = std.posix;

pub const SnapshotError = error{
    SnapshotFailed,
    InvalidSnapshot,
    IoError,
};

pub const SnapshotMetadata = struct {
    id: []const u8,
    timestamp: i64,
    memory_size: u64,
    memory_mb: u32,
    vcpus: u32,
};

pub const SNAPSHOT_DIR = "/var/lib/coco/snapshots";

pub fn createSnapshot(id: []const u8, _: [*]u8, _: u64) !void {
    const snap_dir = try std.fmt.allocPrint(std.heap.page_allocator, "{s}/{s}", .{ SNAPSHOT_DIR, id });
    defer std.heap.page_allocator.free(snap_dir);

    try std.fs.cwd().makeDir(snap_dir);
}

pub fn loadSnapshot(id: []const u8) !SnapshotMetadata {
    return SnapshotMetadata{
        .id = id,
        .timestamp = std.time.timestamp(),
        .memory_size = 0,
        .memory_mb = 512,
        .vcpus = 2,
    };
}

pub fn deleteSnapshot(id: []const u8) void {
    const snap_dir = std.fmt.allocPrint(std.heap.page_allocator, "{s}/{s}", .{ SNAPSHOT_DIR, id }) catch return;
    defer std.heap.page_allocator.free(snap_dir);

    std.fs.deleteTreeAbsolute(snap_dir) catch {};
}
