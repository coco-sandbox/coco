// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;
const linux = std.os.linux;

const ReflinkError = error{
    CopyFileRangeFailed,
    IoctlFailed,
    NotSupported,
};

pub const ReflinkOptions = struct {
    sparse: bool = true,
    compression: ?[]const u8 = null,
};

pub fn reflinkCopy(src_fd: posix.fd_t, dst_fd: posix.fd_t, src_off: u64, dst_off: u64, length: u64) !u64 {
    var from = linux.off_t{ .pos = src_off };
    var to = linux.off_t{ .pos = dst_off };

    const result = linux.copy_file_range(src_fd, &from, dst_fd, &to, length, 0);
    if (result < 0) {
        return ReflinkError.CopyFileRangeFailed;
    }

    return @intCast(result);
}

pub fn reflinkFile(src_path: []const u8, dst_path: []const u8) !void {
    const src_fd = try posix.open(src_path, posix.O_RDONLY, 0);
    defer posix.close(src_fd);

    const dst_fd = try posix.open(dst_path, posix.O_CREAT | posix.O_WRONLY, 0o644);
    defer posix.close(dst_fd);

    const stat = try posix.fstat(src_fd);
    const size = @as(u64, @intCast(stat.size));
    const copied = try reflinkCopy(src_fd, dst_fd, 0, 0, size);

    if (copied != size) {
        return ReflinkError.CopyFileRangeFailed;
    }
}

pub fn supportsReflink(path: []const u8) bool {
    const fd = posix.open(path, posix.O_RDONLY, 0) catch return false;
    defer posix.close(fd);

    var stat: posix.stat = undefined;
    if (posix.fstat(fd, &stat) != 0) return false;

    if (stat.blksize == 0) return false;

    return true;
}

pub fn isBtrfs(filesystem: []const u8) bool {
    return std.mem.eql(u8, filesystem, "btrfs");
}

pub fn getFilesystemType(path: []const u8) ?[]u8 {
    _ = path;
    return null;
}

pub fn deduplicateFiles(files: [][]const u8) !void {
    if (files.len < 2) return;

    for (files[1..]) |file| {
        reflinkFile(files[0], file) catch {};
    }
}
