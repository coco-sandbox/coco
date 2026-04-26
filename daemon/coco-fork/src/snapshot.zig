// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

const SnapshotError = error{
    CreateFailed,
    RestoreFailed,
    NotFound,
    IoError,
};

pub const Snapshot = struct {
    id: []u8,
    path: []u8,
    size: u64,
    created_at: i64,

    pub fn format(self: Snapshot, writer: anytype) !void {
        try writer.print("Snapshot{id={s}, path={s}, size={d}}", .{
            self.id, self.path, self.size,
        });
    }
};

pub const SnapshotManager = struct {
    snapshot_dir: []u8,
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator, snapshot_dir: []u8) SnapshotManager {
        return SnapshotManager{
            .snapshot_dir = snapshot_dir,
            .allocator = allocator,
        };
    }

    pub fn createSnapshot(self: *SnapshotManager, vm_id: []const u8) !Snapshot {
        const snapshot_id = try std.fmt.allocPrint(self.allocator, "snap-{s}-{d}", .{
            vm_id, std.time.timestamp(),
        });

        const snapshot_path = try std.fmt.allocPrint(self.allocator, "{s}/{s}", .{
            self.snapshot_dir, snapshot_id,
        });

        try std.fs.cwd().makeDir(snapshot_path);

        return Snapshot{
            .id = snapshot_id,
            .path = snapshot_path,
            .size = 0,
            .created_at = std.time.timestamp(),
        };
    }

    pub fn deleteSnapshot(self: *SnapshotManager, snapshot_id: []const u8) !void {
        const snapshot_path = try std.fmt.allocPrint(self.allocator, "{s}/{s}", .{
            self.snapshot_dir, snapshot_id,
        });

        try std.fs.cwd().deleteTree(snapshot_path);
    }

    pub fn listSnapshots(self: *SnapshotManager) ![]Snapshot {
        var dir = try std.fs.cwd().openDir(self.snapshot_dir, .{});
        defer dir.close();

        var snapshots = std.ArrayList(Snapshot).init(self.allocator);
        defer snapshots.deinit();

        var iterator = dir.iterate();
        while (try iterator.next()) |entry| {
            if (entry.kind == .Directory) {
                try snapshots.append(Snapshot{
                    .id = try self.allocator.dupe(u8, entry.name),
                    .path = try std.fmt.allocPrint(self.allocator, "{s}/{s}", .{
                        self.snapshot_dir, entry.name,
                    }),
                    .size = 0,
                    .created_at = std.time.timestamp(),
                });
            }
        }

        return snapshots.toOwnedSlice();
    }
};

pub fn saveMemorySnapshot(path: []const u8, memory_fd: posix.fd_t) !u64 {
    const snapshot_file = try posix.open(path, posix.O.CREAT | posix.O_WRONLY, 0o644);
    defer posix.close(snapshot_file);

    var buf: [65536]u8 = undefined;
    var total: u64 = 0;

    while (true) {
        const n = posix.read(memory_fd, &buf) catch {
            return SnapshotError.IoError;
        };
        if (n == 0) break;

        const written = posix.write(snapshot_file, buf[0..n]) catch {
            return SnapshotError.IoError;
        };
        total += written;
    }

    return total;
}

pub fn restoreMemorySnapshot(path: []const u8, memory_fd: posix.fd_t) !u64 {
    const snapshot_file = try posix.open(path, posix.O_RDONLY, 0);
    defer posix.close(snapshot_file);

    var buf: [65536]u8 = undefined;
    var total: u64 = 0;

    while (true) {
        const n = posix.read(snapshot_file, &buf) catch {
            return SnapshotError.IoError;
        };
        if (n == 0) break;

        const written = posix.write(memory_fd, buf[0..n]) catch {
            return SnapshotError.IoError;
        };
        total += written;
    }

    return total;
}
