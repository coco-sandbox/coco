// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

pub const Process = struct {
    pid: posix.pid_t,
    exit_code: ?u8 = null,
    term_signal: ?c_int = null,
    status: ProcessStatus = .running,

    pub const ProcessStatus = enum {
        running,
        stopped,
        exited,
    };
};

pub const ProcessManager = struct {
    processes: std.AutoHashMap(posix.pid_t, *Process),
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator) ProcessManager {
        return ProcessManager{
            .processes = std.AutoHashMap(posix.pid_t, *Process).init(allocator),
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *ProcessManager) void {
        self.processes.deinit();
    }

    pub fn add(self: *ProcessManager, pid: posix.pid_t) !*Process {
        const process = try self.allocator.create(Process);
        process.* = Process{ .pid = pid };
        try self.processes.put(pid, process);
        return process;
    }

    pub fn remove(self: *ProcessManager, pid: posix.pid_t) void {
        if (self.processes.get(pid)) |process| {
            self.allocator.destroy(process);
            self.processes.remove(pid);
        }
    }

    pub fn get(self: *ProcessManager, pid: posix.pid_t) ?*Process {
        return self.processes.get(pid);
    }

    pub fn reap(self: *ProcessManager) void {
        var status: posix.wait.Status = undefined;
        while (true) {
            const result = posix.waitpid(-1, &status, .WNOHANG);
            if (result == 0 or result == -1) break;

            const pid = result;
            if (self.processes.get(pid)) |process| {
                if (status.exited()) {
                    process.exit_code = status.code();
                    process.status = .exited;
                } else if (status.signal()) |sig| {
                    process.term_signal = sig;
                    process.status = .exited;
                }
            }
        }
    }

    pub fn killAll(self: *ProcessManager) void {
        var iterator = self.processes.keyIterator();
        while (iterator.next()) |pid| {
            posix.kill(pid.*, posix.SIG.KILL) catch {};
        }
    }

    pub fn count(self: *ProcessManager) usize {
        return self.processes.size;
    }
};

pub fn createProcessGroup() !posix.pid_t {
    return posix.setsid();
}

pub fn joinProcessGroup(pgid: posix.pid_t) void {
    posix.setpgid(0, pgid) catch {};
}
