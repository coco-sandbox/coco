// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Checkpoint restoration operations.

const std = @import("std");
const posix = std.posix;
const sc = @import("syscall.zig");
const checkpoint = @import("checkpoint.zig");

pub const RestoreError = error{
    RestoreFailed,
    CheckpointNotFound,
    InvalidCheckpoint,
    OutOfMemory,
};

pub const RestoreResult = struct {
    pid: u32,
    vsock_cid: u32,
    duration_ms: u32,
};

pub fn restoreFromSnapshot(
    id: []const u8,
    mem_ptr: [*]u8,
    mem_size: u64,
    allocator: std.mem.Allocator,
) !RestoreResult {
    const start = sc.nanoTimestamp();

    var checkpoint_mgr = checkpoint.CheckpointManager.init(allocator);
    defer checkpoint_mgr.deinit();

    const metadata = checkpoint_mgr.restoreCheckpoint(id, mem_ptr);

    if (metadata.id.len == 0) {
        return RestoreError.CheckpointNotFound;
    }

    const checkpoint_dir = std.fmt.allocPrint(allocator, "{any}/{any}", .{ checkpoint.SNAPSHOT_DIR, id }) catch return RestoreError.OutOfMemory;
    defer allocator.free(checkpoint_dir);

    const cpu_path = std.fmt.allocPrint(allocator, "{any}/cpu.bin", .{checkpoint_dir}) catch return RestoreError.OutOfMemory;
    defer allocator.free(cpu_path);
    checkpoint_mgr.loadCpuState(cpu_path);

    const devices_path = std.fmt.allocPrint(allocator, "{any}/devices.bin", .{checkpoint_dir}) catch return RestoreError.OutOfMemory;
    defer allocator.free(devices_path);
    checkpoint_mgr.loadDeviceState(devices_path);

    const duration = @as(u64, sc.nanoTimestamp() - start);
    const duration_ms = @as(u32, @intCast(@divTrunc(duration, 1_000_000)));

    return RestoreResult{
        .pid = 0,
        .vsock_cid = 0,
        .duration_ms = duration_ms,
    };
}

pub fn restoreWithNewId(
    original_id: []const u8,
    new_id: []const u8,
    new_cid: u32,
    mem_ptr: [*]u8,
    mem_size: u64,
    allocator: std.mem.Allocator,
) !RestoreResult {
    var checkpoint_mgr = checkpoint.CheckpointManager.init(allocator);
    defer checkpoint_mgr.deinit();

    const metadata = checkpoint_mgr.restoreCheckpoint(original_id, mem_ptr);

    if (metadata.id.len == 0) {
        return RestoreError.CheckpointNotFound;
    }

    const new_checkpoint_dir = std.fmt.allocPrint(allocator, "{any}/{any}", .{ checkpoint.SNAPSHOT_DIR, new_id }) catch return RestoreError.OutOfMemory;
    defer allocator.free(new_checkpoint_dir);

    std.fs.cwd().makeDir(new_checkpoint_dir) catch return RestoreError.RestoreFailed;

    const old_checkpoint_dir = std.fmt.allocPrint(allocator, "{any}/{any}", .{ checkpoint.SNAPSHOT_DIR, original_id }) catch return RestoreError.OutOfMemory;
    defer allocator.free(old_checkpoint_dir);

    var old_dir = std.fs.openDirAbsolute(old_checkpoint_dir, .{ .iterate = true }) catch return RestoreError.RestoreFailed;
    defer old_dir.close();

    var iter = old_dir.iterate();
    while (iter.next() catch null) |entry| {
        const src = std.fmt.allocPrint(allocator, "{any}/{any}", .{ old_checkpoint_dir, entry.name }) catch continue;
        defer allocator.free(src);

        const dst = std.fmt.allocPrint(allocator, "{any}/{any}", .{ new_checkpoint_dir, entry.name }) catch continue;
        defer allocator.free(dst);

        switch (entry.kind) {
            .file => {
                std.fs.copyFileAbsolute(src, dst, .{}) catch {};
            },
            .directory => {
                std.fs.cwd().makeDir(dst) catch {};
            },
            else => {},
        }
    }

    return RestoreResult{
        .pid = 0,
        .vsock_cid = new_cid,
        .duration_ms = 0,
    };
}

pub fn listAvailableSnapshots(allocator: std.mem.Allocator) [][]const u8 {
    var checkpoint_mgr = checkpoint.CheckpointManager.init(allocator);
    return checkpoint_mgr.listCheckpoints();
}
