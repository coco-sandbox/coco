// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

const CloneError = error{
    CloneFailed,
    WaitFailed,
    SetnsFailed,
};

pub const CloneOptions = struct {
    stack_size: usize = 65536,
    flags: u64 = 0,
};

pub const CloneResult = struct {
    pid: posix.pid_t,
    success: bool,
};

pub fn cloneProcess(opts: CloneOptions) CloneError!CloneResult {
    var stack: [65536]u8 = undefined;

    const flags = opts.flags |
        posix.CLONE.NEWNS |
        posix.CLONE.NEWUTS |
        posix.CLONE.NEWIPC |
        posix.CLONE.NEWPID |
        posix.CLONE.NEWNET;

    const pid = posix.clone(&stack, flags, null);
    if (pid < 0) {
        return CloneError.CloneFailed;
    }

    return CloneResult{
        .pid = pid,
        .success = true,
    };
}

pub fn createNamespace() !void {
    _ = try posix.unshare(posix.CLONE.NEWNS);
}

pub fn setNamespace(path: []const u8) !void {
    const fd = try posix.open(path, posix.O.RDONLY, 0);
    defer posix.close(fd);

    try posix.setns(fd, 0);
}

pub fn getNamespacePath(pid: posix.pid_t, ns_type: []const u8) ![]u8 {
    const path = try std.fmt.allocPrint(std.heap.c_allocator, "/proc/{d}/ns/{s}", .{ pid, ns_type });
    return path;
}

pub fn isInNamespace(pid: posix.pid_t, ns_type: []const u8) bool {
    const current_path = getNamespacePath(0, ns_type) catch return false;
    defer std.heap.c_allocator.free(current_path);

    const target_path = getNamespacePath(pid, ns_type) catch return false;
    defer std.heap.c_allocator.free(target_path);

    return std.mem.eql(u8, current_path, target_path);
}
