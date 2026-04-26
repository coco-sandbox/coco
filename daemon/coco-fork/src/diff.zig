// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

const DiffError = error{
    OpenFailed,
    ReadFailed,
    WriteFailed,
    InvalidSnapshot,
};

pub const DiffRange = struct {
    offset: u64,
    length: u64,
    data: ?[]u8 = null,
};

pub const Diff = struct {
    ranges: []DiffRange,
    total_changed: u64,

    pub fn deinit(self: *Diff, allocator: std.mem.Allocator) void {
        for (self.ranges) |range| {
            if (range.data) |data| {
                allocator.free(data);
            }
        }
        allocator.free(self.ranges);
    }
};

pub const DiffOptions = struct {
    block_size: usize = 4096,
    compression: bool = false,
};

pub fn computeDiff(old_path: []const u8, new_path: []const u8, allocator: std.mem.Allocator, opts: DiffOptions) !Diff {
    const old_fd = posix.open(old_path, posix.O_RDONLY, 0) catch {
        return DiffError.OpenFailed;
    };
    defer posix.close(old_fd);

    const new_fd = posix.open(new_path, posix.O_RDONLY, 0) catch {
        return DiffError.OpenFailed;
    };
    defer posix.close(new_fd);

    var old_stat = try posix.fstat(old_fd);
    var new_stat = try posix.fstat(new_fd);

    const old_size = @intCast(old_stat.size);
    const new_size = @intCast(new_stat.size);

    var ranges = std.ArrayList(DiffRange).init(allocator);
    defer ranges.deinit();

    var offset: u64 = 0;
    var total_changed: u64 = 0;

    const max_size = @max(old_size, new_size);
    while (offset < max_size) : (offset += opts.block_size) {
        var old_buf: [4096]u8 = undefined;
        var new_buf: [4096]u8 = undefined;

        const old_read = posix.pread(old_fd, &old_buf, opts.block_size, @intCast(offset)) catch {
            continue;
        };
        const new_read = posix.pread(new_fd, &new_buf, opts.block_size, @intCast(offset)) catch {
            continue;
        };

        if (old_read != new_read or !std.mem.eql(u8, old_buf[0..old_read], new_buf[0..new_read])) {
            const chunk_size = @max(old_read, new_read);
            var data = try allocator.alloc(u8, chunk_size);

            @memcpy(data[0..chunk_size], new_buf[0..chunk_size]);

            try ranges.append(DiffRange{
                .offset = offset,
                .length = chunk_size,
                .data = data,
            });

            total_changed += chunk_size;
        }
    }

    return Diff{
        .ranges = try ranges.toOwnedSlice(),
        .total_changed = total_changed,
    };
}

pub fn applyDiff(base_path: []const u8, diff: *Diff, output_path: []const u8) !void {
    const base_fd = try posix.open(base_path, posix.O_RDONLY, 0);
    defer posix.close(base_fd);

    const output_fd = try posix.open(output_path, posix.O_CREAT | posix.O_WRONLY, 0o644);
    defer posix.close(output_fd);

    for (diff.ranges) |range| {
        var buf: [4096]u8 = undefined;

        const old_read = posix.pread(base_fd, &buf, range.length, range.offset) catch {
            continue;
        };

        _ = posix.pwrite(output_fd, buf[0..old_read], range.offset) catch {
            return DiffError.WriteFailed;
        };

        if (range.data) |data| {
            _ = posix.pwrite(output_fd, data, range.offset) catch {
                return DiffError.WriteFailed;
            };
        }
    }
}

pub fn computeIncrementalDiff(old_path: []const u8, new_path: []const u8, allocator: std.mem.Allocator) !Diff {
    return computeDiff(old_path, new_path, allocator, DiffOptions{});
}
