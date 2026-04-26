// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cocod — Guest agent running inside MicroVM as PID 1.
//! Listens on vsock for exec commands from the host.
//! Streams stdout/stderr back via vsock connection.
//! Falls back to TCP for development environments.

const std = @import("std");
const posix = std.posix;
const linux = std.os.linux;

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_VSOCK_PORT: u16 = 4747;
const RECV_BUF_SIZE: usize = 65536;
const CHUNK_SIZE: usize = 32768;

// Stream types for vsock protocol (host → cocod)
const CMD_EXEC: u32 = 1;

// Stream types for vsock protocol (cocod → host)
const STREAM_STDOUT: u32 = 1;
const STREAM_STDERR: u32 = 2;
const STREAM_EXIT: u32 = 3;

// =============================================================================
// Global State
// =============================================================================

var allocator = std.heap.c_allocator;
var running = true;
var child_pid: posix.pid_t = 0;

// =============================================================================
// Logging
// =============================================================================

fn log(comptime fmt: []const u8, args: anytype) void {
    std.debug.print("[cocod] " ++ fmt ++ "\n", args);
}

fn logError(comptime fmt: []const u8, args: anytype) void {
    std.debug.print("[codod] ERROR: " ++ fmt ++ "\n", args);
}

// =============================================================================
// Signal Handling (PID 1 responsibilities)
// =============================================================================

fn handleSigterm(sig_num: linux.SIG) callconv(.c) void {
    _ = sig_num;
    log("Received SIGTERM, initiating graceful shutdown", .{});
    running = false;

    if (child_pid > 0) {
        log("Forwarding SIGTERM to child process {d}", .{child_pid});
        posix.kill(child_pid, posix.SIG.TERM) catch {};
    }
}

fn handleSigchld(sig_num: linux.SIG) callconv(.c) void {
    _ = sig_num;
    while (true) {
        var status: i32 = 0;
        const pid = linux.waitpid(-1, &status, linux.W.NOHANG);
        if (pid <= 0) break;
        if (pid == child_pid) child_pid = 0;
    }
}

fn setupSignalHandlers() void {
    const empty_set: linux.sigset_t = std.mem.zeroes(linux.sigset_t);

    var sa_term: linux.Sigaction = .{
        .handler = .{ .handler = handleSigterm },
        .mask = empty_set,
        .flags = 0,
    };
    _ = linux.sigaction(posix.SIG.TERM, &sa_term, null);

    var sa_int: linux.Sigaction = .{
        .handler = .{ .handler = handleSigterm },
        .mask = empty_set,
        .flags = 0,
    };
    _ = linux.sigaction(posix.SIG.INT, &sa_int, null);

    var sa_hup: linux.Sigaction = .{
        .handler = .{ .handler = handleSigterm },
        .mask = empty_set,
        .flags = 0,
    };
    _ = linux.sigaction(posix.SIG.HUP, &sa_hup, null);

    var sa_pipe: linux.Sigaction = .{
        .handler = .{ .handler = posix.SIG.IGN },
        .mask = empty_set,
        .flags = 0,
    };
    _ = linux.sigaction(posix.SIG.PIPE, &sa_pipe, null);

    var sa_chld: linux.Sigaction = .{
        .handler = .{ .handler = handleSigchld },
        .mask = empty_set,
        .flags = posix.SA.NOCLDSTOP,
    };
    _ = linux.sigaction(posix.SIG.CHLD, &sa_chld, null);
}

// =============================================================================
// Stream Wrapper
// =============================================================================

pub const Stream = struct {
    handle: i32,

    pub fn read(self: Stream, buf: []u8) !usize {
        const rc = linux.read(self.handle, buf.ptr, buf.len);
        if (rc < 0) return error.ReadFailed;
        return @intCast(rc);
    }

    pub fn writeAll(self: Stream, data: []const u8) !void {
        var written: usize = 0;
        while (written < data.len) {
            const rc = linux.write(self.handle, data[written..].ptr, data.len - written);
            if (rc < 0) return error.WriteFailed;
            written += @intCast(rc);
        }
    }

    pub fn readAtLeast(self: Stream, buf: []u8, min_len: usize) !usize {
        var total: usize = 0;
        while (total < min_len) {
            const n = try self.read(buf[total..]);
            if (n == 0) return error.EndOfStream;
            total += n;
        }
        return total;
    }

    pub fn close(self: Stream) void {
        _ = linux.close(self.handle);
    }
};

// =============================================================================
// VSock Communication (TCP fallback)
// =============================================================================

const AF_VSOCK: u32 = 40;

const SockaddrVm = extern struct {
    svm_family: u16 = 40,
    svm_reserved1: u16 = 0,
    svm_port: u32,
    svm_cid: u32,
    svm_flags: u8 = 0,
    svm_zero: [3]u8 = .{ 0, 0, 0 },
};

const sockaddr_in = extern struct {
    family: u16,
    port: u16,
    addr: u32,
    zero: [8]u8,
};

fn connectVsock(cid: u32, port: u32) !Stream {
    const fd_usize = linux.socket(AF_VSOCK, linux.SOCK.STREAM, 0);
    if (fd_usize == std.math.maxInt(usize)) return error.SocketFailed;
    const fd: i32 = @intCast(fd_usize);
    errdefer _ = linux.close(fd);
    const addr = SockaddrVm{ .svm_port = port, .svm_cid = cid };
    const rc = linux.connect(fd, @ptrCast(&addr), @sizeOf(SockaddrVm));
    if (rc != 0) return error.ConnectFailed;
    return Stream{ .handle = fd };
}

fn connectToHost(port: u16) !Stream {
    log("Trying VSock CID=2 port={d}", .{port});
    if (connectVsock(2, @as(u32, port))) |stream| {
        log("Connected via VSock", .{});
        return stream;
    } else |_| {
        log("VSock failed, trying TCP fallback", .{});
        const fd_usize = linux.socket(linux.AF.INET, linux.SOCK.STREAM, 0);
        if (fd_usize == std.math.maxInt(usize)) return error.SocketFailed;
        const fd: i32 = @intCast(fd_usize);
        errdefer _ = linux.close(fd);
        const addr = sockaddr_in{
            .family = linux.AF.INET,
            .port = std.mem.nativeToBig(u16, port),
            .addr = 0x0100007f,
            .zero = .{ 0, 0, 0, 0, 0, 0, 0, 0 },
        };
        const rc = linux.connect(fd, @ptrCast(&addr), @sizeOf(sockaddr_in));
        if (rc != 0) return error.ConnectFailed;
        return Stream{ .handle = fd };
    }
}

/// sendChunk sends a stream chunk to host with proper framing
fn sendChunk(sock: Stream, stream_type: u32, data: []const u8) !void {
    // Length-prefixed message: [4-byte stream_type][4-byte payload_len][payload]
    var stream_type_val: u32 = stream_type;
    var payload_len_val: u32 = @as(u32, @intCast(data.len));

    try sock.writeAll(std.mem.asBytes(&stream_type_val));
    try sock.writeAll(std.mem.asBytes(&payload_len_val));
    if (data.len > 0) {
        try sock.writeAll(data);
    }
}

/// sendExit sends the exit code to host
fn sendExit(sock: Stream, code: u32) !void {
    // Exit message: [stream_type=3][payload_len=4][exit_code]
    var stream_type_val: u32 = STREAM_EXIT;
    var payload_len_val: u32 = 4;
    var exit_code_val: u32 = code;

    try sock.writeAll(std.mem.asBytes(&stream_type_val));
    try sock.writeAll(std.mem.asBytes(&payload_len_val));
    try sock.writeAll(std.mem.asBytes(&exit_code_val));
}

fn forkExecCommand(
    cmd: []const u8,
    args_raw: []const u8,
    env_raw: []const u8,
    workdir: []const u8,
    stdout_w: i32,
    stderr_w: i32,
) !posix.pid_t {
    const pid_rc = linux.fork();
    const pid_sr: isize = @bitCast(pid_rc);
    if (pid_sr < 0) return error.ForkFailed;
    const pid: posix.pid_t = @intCast(pid_sr);

    if (pid == 0) {
        _ = linux.dup2(stdout_w, 1);
        _ = linux.dup2(stderr_w, 2);

        if (workdir.len > 0) {
            var wd_buf: [4096]u8 = undefined;
            if (workdir.len < wd_buf.len) {
                @memcpy(wd_buf[0..workdir.len], workdir);
                wd_buf[workdir.len] = 0;
                _ = linux.chdir(@ptrCast(&wd_buf));
            }
        }

        var argv_storage: [128][*:0]const u8 = undefined;
        var argv_buf: [4096]u8 = undefined;
        var bp: usize = 0;
        var ai: usize = 0;

        if (cmd.len + 1 < argv_buf.len - bp) {
            @memcpy(argv_buf[bp..][0..cmd.len], cmd);
            argv_buf[bp + cmd.len] = 0;
            argv_storage[ai] = @ptrCast(&argv_buf[bp]);
            bp += cmd.len + 1;
            ai += 1;
        }

        var i: usize = 0;
        while (i < args_raw.len and ai < 127) {
            var j = i;
            while (j < args_raw.len and args_raw[j] != 0) : (j += 1) {}
            if (j > i and bp + (j - i) + 1 < argv_buf.len) {
                @memcpy(argv_buf[bp..][0..(j - i)], args_raw[i..j]);
                argv_buf[bp + (j - i)] = 0;
                argv_storage[ai] = @ptrCast(&argv_buf[bp]);
                bp += (j - i) + 1;
                ai += 1;
            }
            i = j + 1;
        }

        var envp_storage: [128][*:0]const u8 = undefined;
        var envp_buf: [4096]u8 = undefined;
        var ebp: usize = 0;
        var ei: usize = 0;

        i = 0;
        while (i < env_raw.len and ei < 127) {
            var j = i;
            while (j < env_raw.len and env_raw[j] != 0) : (j += 1) {}
            if (j > i and ebp + (j - i) + 1 < envp_buf.len) {
                @memcpy(envp_buf[ebp..][0..(j - i)], env_raw[i..j]);
                envp_buf[ebp + (j - i)] = 0;
                envp_storage[ei] = @ptrCast(&envp_buf[ebp]);
                ebp += (j - i) + 1;
                ei += 1;
            }
            i = j + 1;
        }

        const argv_term: ?[*:0]const u8 = null;
        const envp_term: ?[*:0]const u8 = null;
        var argv_final: [128]?[*:0]const u8 = undefined;
        var envp_final: [128]?[*:0]const u8 = undefined;
        for (0..ai) |k| argv_final[k] = argv_storage[k];
        argv_final[ai] = argv_term;
        for (0..ei) |k| envp_final[k] = envp_storage[k];
        envp_final[ei] = envp_term;

        _ = linux.execve(argv_storage[0], @ptrCast(&argv_final), @ptrCast(&envp_final));
        linux.exit(127);
        unreachable;
    }

    return pid;
}

fn executeStructuredCommand(sock: Stream, cmd: []const u8, args_raw: []const u8, env_raw: []const u8, workdir: []const u8) !void {
    var stdout_pipe: [2]i32 = undefined;
    var stderr_pipe: [2]i32 = undefined;
    if (@as(isize, @bitCast(linux.pipe(&stdout_pipe))) < 0) return error.PipeFailed;
    if (@as(isize, @bitCast(linux.pipe(&stderr_pipe))) < 0) return error.PipeFailed;

    const pid = try forkExecCommand(cmd, args_raw, env_raw, workdir, stdout_pipe[1], stderr_pipe[1]);
    child_pid = pid;
    _ = linux.close(stdout_pipe[1]);
    _ = linux.close(stderr_pipe[1]);

    var stdout_buf: [CHUNK_SIZE]u8 = undefined;
    var stderr_buf: [CHUNK_SIZE]u8 = undefined;
    var stdout_done = false;
    var stderr_done = false;

    while (!stdout_done or !stderr_done) {
        if (!stdout_done) {
            const n_rc = linux.read(stdout_pipe[0], &stdout_buf, stdout_buf.len);
            const n_sr: isize = @bitCast(n_rc);
            if (n_sr <= 0) {
                stdout_done = true;
            } else {
                try sendChunk(sock, STREAM_STDOUT, stdout_buf[0..@intCast(n_sr)]);
            }
        }
        if (!stderr_done) {
            const n_rc = linux.read(stderr_pipe[0], &stderr_buf, stderr_buf.len);
            const n_sr: isize = @bitCast(n_rc);
            if (n_sr <= 0) {
                stderr_done = true;
            } else {
                try sendChunk(sock, STREAM_STDERR, stderr_buf[0..@intCast(n_sr)]);
            }
        }
    }
    _ = linux.close(stdout_pipe[0]);
    _ = linux.close(stderr_pipe[0]);

    var status: i32 = 0;
    _ = linux.waitpid(pid, &status, 0);
    child_pid = 0;

    var code: u32 = 0;
    if ((status & 0x7f) == 0) {
        code = @intCast((status >> 8) & 0xff);
    } else {
        code = 128 + @as(u32, @intCast(status & 0x7f));
    }
    try sendExit(sock, code);
}

/// recvMessage receives a length-prefixed message from vsock
fn recvMessage(sock: Stream, buf: []u8) ![]u8 {
    // Read 4-byte length header
    var len_buf: [4]u8 = undefined;
    _ = try sock.readAtLeast(&len_buf, 4);
    const payload_len = std.mem.readInt(u32, &len_buf, .little);

    if (payload_len > buf.len) {
        return error.MessageTooLarge;
    }

    // Read payload
    _ = try sock.readAtLeast(buf[0..payload_len], payload_len);
    return buf[0..payload_len];
}

// =============================================================================
// Command Execution with Streaming
// =============================================================================

/// executeStreamingCommand executes a command and streams stdout/stderr back
fn executeStreamingCommand(sock: Stream, cmdline: []const u8) !void {
    log("Executing: {s}", .{cmdline});

    // Parse command line into argv
    var args = std.ArrayList([]const u8).init(allocator);
    defer args.deinit();

    var it = std.mem.splitSequence(u8, cmdline, " ");
    while (it.next()) |arg| {
        if (arg.len > 0) {
            try args.append(try allocator.dupe(u8, arg));
        }
    }

    if (args.items.len == 0) {
        try sendChunk(sock, STREAM_STDERR, "No command provided\n");
        try sendExit(sock, 1);
        return;
    }

    // Build argv slice
    const argv = try args.toOwnedSlice();
    defer allocator.free(argv);

    // Create child process
    var child = std.process.Child.init(argv, allocator);
    child.stdout_behavior = .Pipe;
    child.stderr_behavior = .Pipe;

    // Spawn
    try child.spawn();
    child_pid = child.id;
    log("Child process spawned with PID {d}", .{child_pid});

    // Stream stdout/stderr in parallel with reading
    var stdout_done = false;
    var stderr_done = false;
    var exit_code: u32 = 0;

    var stdout_buf: [CHUNK_SIZE]u8 = undefined;
    var stderr_buf: [CHUNK_SIZE]u8 = undefined;

    while (!stdout_done or !stderr_done) {
        if (!stdout_done) {
            if (child.stdout) |stdout| {
                const n = stdout.read(&stdout_buf) catch |e| {
                    logError("stdout read error: {}", .{e});
                    stdout_done = true;
                    break;
                };
                if (n == 0) {
                    stdout_done = true;
                } else if (n > 0) {
                    try sendChunk(sock, STREAM_STDOUT, stdout_buf[0..n]);
                }
            } else {
                stdout_done = true;
            }
        }

        if (!stderr_done) {
            if (child.stderr) |stderr| {
                const n = stderr.read(&stderr_buf) catch |e| {
                    logError("stderr read error: {}", .{e});
                    stderr_done = true;
                    break;
                };
                if (n == 0) {
                    stderr_done = true;
                } else if (n > 0) {
                    try sendChunk(sock, STREAM_STDERR, stderr_buf[0..n]);
                }
            } else {
                stderr_done = true;
            }
        }

        // Small sleep to avoid busy looping when pipes are empty but process is still running
        if (!stdout_done or !stderr_done) {
            std.time.sleep(10 * std.time.ns_per_ms);
        }
    }

    // Wait for process to finish and get exit code
    const term = child.wait() catch |e| {
        logError("Failed to wait for child: {}", .{e});
        exit_code = 255;
        child_pid = 0;
        try sendExit(sock, exit_code);
        return;
    };

    child_pid = 0;

    switch (term) {
        .Exited => |code| {
            exit_code = @as(u32, @intCast(code));
            log("Child exited with code {d}", .{code});
        },
        .Signal => |sig| {
            exit_code = 128 + @as(u32, @intCast(sig));
            log("Child killed by signal {d}", .{sig});
        },
        .Stopped => |sig| {
            exit_code = 128 + @as(u32, @intCast(sig));
            log("Child stopped by signal {d}", .{sig});
        },
        .Unknown => |code| {
            exit_code = 128 + @as(u32, @intCast(code));
            log("Child exited with unknown status {d}", .{code});
        },
    }

    try sendExit(sock, exit_code);
}

// =============================================================================
// Main Loop
// =============================================================================

/// runMainLoop listens for commands from host and executes them
fn runMainLoop(sock: Stream) !void {
    log("Main loop started", .{});

    var recv_buf: [RECV_BUF_SIZE]u8 = undefined;
    var pollfd: [1]posix.pollfd = [1]posix.pollfd{.{
        .fd = sock.handle,
        .events = posix.POLL.IN,
        .revents = 0,
    }};

    while (running) {
        // Poll with 100ms timeout to allow checking running flag
        const poll_result = posix.poll(&pollfd, 100) catch |e| {
            logError("Poll error: {}", .{e});
            break;
        };

        if (poll_result == 0) {
            // Timeout, check running flag and continue
            continue;
        }

        if (pollfd[0].revents & posix.POLL.HUP != 0) {
            log("Connection closed (HUP)", .{});
            break;
        }

        if (pollfd[0].revents & posix.POLL.ERR != 0) {
            logError("Socket error", .{});
            break;
        }

        if (pollfd[0].revents & posix.POLL.IN == 0) {
            continue;
        }

        // Socket is readable, try to receive
        const msg = recvMessage(sock, &recv_buf) catch |e| {
            logError("Receive error: {}", .{e});
            break;
        };

        if (msg.len < 4) {
            logError("Invalid message length: {d}", .{msg.len});
            continue;
        }

        // Parse message type (first 4 bytes)
        const msg_type = std.mem.readInt(u32, msg[0..4], .little);
        const payload = msg[4..];

        switch (msg_type) {
            CMD_EXEC => {
                if (payload.len < 20) {
                    logError("CMD_EXEC payload too small: {d}", .{payload.len});
                    continue;
                }
                _ = std.mem.readInt(u32, payload[0..4], .little);
                const cmd_len = std.mem.readInt(u32, payload[4..8], .little);
                const args_len = std.mem.readInt(u32, payload[8..12], .little);
                const env_len = std.mem.readInt(u32, payload[12..16], .little);
                const workdir_len = std.mem.readInt(u32, payload[16..20], .little);
                const base: usize = 20;
                const cmd = payload[base..][0..cmd_len];
                const args_raw = payload[base + cmd_len ..][0..args_len];
                const env_raw = payload[base + cmd_len + args_len ..][0..env_len];
                const workdir = payload[base + cmd_len + args_len + env_len ..][0..workdir_len];
                executeStructuredCommand(sock, cmd, args_raw, env_raw, workdir) catch |e| {
                    logError("executeStructuredCommand failed: {}", .{e});
                };
            },
            else => {
                log("Unknown message type: {d}", .{msg_type});
            },
        }
    }

    log("Main loop ended", .{});
}

// =============================================================================
// PID 1 Responsibilities
// =============================================================================

fn setupPid1() void {
    _ = linux.mkdir("/proc", 0o755);
    _ = linux.mkdir("/sys", 0o755);
    _ = linux.mkdir("/dev", 0o755);
    _ = linux.mkdir("/tmp", 0o755);
    _ = linux.mkdir("/var", 0o755);
    _ = linux.mkdir("/var/log", 0o755);

    const MS_NOEXEC: u64 = 8;
    const MS_NOSUID: u64 = 2;
    const MS_NODEV: u64 = 4;
    _ = linux.mount("proc", "/proc", "proc", MS_NOEXEC | MS_NOSUID | MS_NODEV, 0);
    _ = linux.mount("sysfs", "/sys", "sysfs", MS_NOEXEC | MS_NOSUID | MS_NODEV, 0);
    _ = linux.mount("devtmpfs", "/dev", "devtmpfs", 0, 0);
    _ = linux.mount("tmpfs", "/tmp", "tmpfs", 0, 0);
}

fn ensureDir(path: []const u8) !void {
    std.fs.cwd().access(path, .{}) catch {
        log("Creating directory: {s}", .{path});
        try std.fs.cwd().makeDir(path);
    };
}

// =============================================================================
// Main Entry Point
// =============================================================================

pub fn main() !void {
    const vsock_port_val: u16 = DEFAULT_VSOCK_PORT;

    log("Guest agent starting (PID 1)", .{});
    log("VSock port: {d} (VSock preferred, TCP fallback)", .{vsock_port_val});

    setupPid1();

    setupSignalHandlers();

    // Try to connect to host with retries
    var connected = false;
    var retries: u32 = 0;
    const max_retries: u32 = 30;

    while (!connected and retries < max_retries and running) {
        if (connectToHost(vsock_port_val)) |sock| {
            connected = true;
            log("Connected to host", .{});
            defer sock.close();

            runMainLoop(sock) catch |e| {
                logError("Main loop error: {}", .{e});
            };
        } else |e| {
            retries += 1;
            if (retries % 5 == 0) {
                log("Waiting for host (attempt {d}/{d}): {}", .{ retries, max_retries, e });
            }
            std.time.sleep(1 * std.time.ns_per_s);
        }
    }

    if (!connected) {
        logError("Could not connect to host after {d} attempts", .{max_retries});
        return error.HostUnreachable;
    }

    log("Shutdown complete", .{});
}
