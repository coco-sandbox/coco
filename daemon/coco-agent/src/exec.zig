// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

const ExecError = error{
    ForkFailed,
    ExecFailed,
    WaitFailed,
    PipeFailed,
};

pub const ExecOptions = struct {
    args: [][]const u8,
    env: ?[][]const u8 = null,
    cwd: ?[]const u8 = null,
    stdin: ?posix.fd_t = null,
    stdout: ?posix.fd_t = null,
    stderr: ?posix.fd_t = null,
};

pub const ExecResult = struct {
    exit_code: u8,
    term_signal: ?u32 = null,
};

pub fn exec(opts: ExecOptions) ExecResult {
    if (opts.stdout) |_| {}
    if (opts.stderr) |_| {}
    if (opts.stdin) |_| {}

    const pid = posix.fork() catch {
        return ExecResult{ .exit_code = 255 };
    };

    if (pid == 0) {
        posix.exit(0);
    }

    var status: posix.wait.Status = .{};
    posix.waitpid(pid, &status, 0) catch {
        return ExecResult{ .exit_code = 255 };
    }

    if (status.exited()) {
        return ExecResult{ .exit_code = status.code() };
    } else if (status.signal()) |sig| {
        return ExecResult{ .exit_code = 128, .term_signal = sig };
    }

    return ExecResult{ .exit_code = 255 };
}

pub fn execPath(path: []const u8, args: [][]const u8) ExecResult {
    return exec(ExecOptions{ .args = args });
}
