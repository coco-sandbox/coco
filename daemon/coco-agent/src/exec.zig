// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Command execution for coco-agent.

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
    const pid = posix.fork() catch {
        return ExecResult{ .exit_code = 255 };
    };

    if (pid == 0) {
        if (opts.stdin) |fd| {
            _ = posix.dup2(fd, posix.STDOUT_FILENO) catch {};
        }
        if (opts.stdout) |fd| {
            _ = posix.dup2(fd, posix.STDOUT_FILENO) catch {};
        }
        if (opts.stderr) |fd| {
            _ = posix.dup2(fd, posix.STDERR_FILENO) catch {};
        }

        if (opts.cwd) |cwd| {
            posix.chdir(cwd) catch {};
        }

        const envp = if (opts.env) |env| buildEnv(env) else std.process.env;
        posix.execve(opts.args[0], opts.args, envp) catch {};
        posix.exit(127);
    }

    var status: posix.wait.Status = .{};
    const waited = posix.waitpid(pid, &status, 0);
    if (waited < 0) {
        return ExecResult{ .exit_code = 255 };
    }

    if (status.exited()) {
        return ExecResult{ .exit_code = status.code() };
    } else if (status.signal()) |sig| {
        return ExecResult{ .exit_code = 128, .term_signal = sig };
    }

    return ExecResult{ .exit_code = 255 };
}

fn buildEnv(env: [][]const u8) [][*:0]u8 {
    var result: [128][*:0]u8 = undefined;
    var i: usize = 0;
    while (i < env.len and i < 127) : (i += 1) {
        result[i] = env[i].ptr;
    }
    result[i] = null;
    return result;
}

pub fn execPath(path: []const u8, args: [][]const u8) ExecResult {
    var opts = ExecOptions{ .args = args };
    if (path.len > 0) {
        opts.cwd = path;
    }
    return exec(opts);
}
