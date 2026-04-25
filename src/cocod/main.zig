// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cocod — Guest agent running inside MicroVM as PID 1.
//! Listens on vsock for exec commands from the host.
//! Streams stdout/stderr back via vsock connection.

const std = @import("std");
const posix = std.posix;

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_VSOCK_CID: u32 = 2;
const DEFAULT_VSOCK_PORT: u32 = 4747;
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

var gpa = std.heap.GeneralPurposeAllocator(.{}){};
var allocator: std.mem.Allocator = undefined;
var running = true;
var child_pid: posix.pid_t = 0;

// =============================================================================
// Logging
// =============================================================================

fn log(comptime fmt: []const u8, args: anytype) void {
    std.debug.print("[cocod] " ++ fmt ++ "\n", args);
}

fn logError(comptime fmt: []const u8, args: anytype) void {
    std.debug.print("[cocod] ERROR: " ++ fmt ++ "\n", args);
}

// =============================================================================
// Signal Handling (PID 1 responsibilities)
// =============================================================================

fn setupSignalHandlers() void {
    // Setup sigaction for SIGTERM
    var sa_term: posix.struct_sigaction = .{
        .handler = .{ .handler = &handleSigterm },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    posix.sigaction(posix.SIG.TERM, &sa_term, null);

    // Setup sigaction for SIGINT
    var sa_int: posix.struct_sigaction = .{
        .handler = .{ .handler = &handleSigterm },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    posix.sigaction(posix.SIG.INT, &sa_int, null);

    // Setup sigaction for SIGHUP
    var sa_hup: posix.struct_sigaction = .{
        .handler = .{ .handler = &handleSigterm },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    posix.sigaction(posix.SIG.HUP, &sa_hup, null);

    // SIGPIPE: ignore (we handle closed sockets explicitly)
    var sa_pipe: posix.struct_sigaction = .{
        .handler = .{ .handler = posix.SIG.IGN },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    posix.sigaction(posix.SIG.PIPE, &sa_pipe, null);

    // Setup sigaction for child signals
    var sa_chld: posix.struct_sigaction = .{
        .handler = .{ .handler = &handleSigchld },
        .mask = posix.empty_sigset,
        .flags = posix.SA.NOCLDSTOP,
    };
    posix.sigaction(posix.SIG.CHLD, &sa_chld, null);
}

fn handleSigterm(_: c_int) callconv(.C) void {
    log("Received SIGTERM, initiating graceful shutdown", .{});
    running = false;

    // Forward SIGTERM to child process if we have one
    if (child_pid > 0) {
        log("Forwarding SIGTERM to child process {d}", .{child_pid});
        posix.kill(child_pid, posix.SIG.TERM) catch {};
    }
}

fn handleSigchld(_: c_int) callconv(.C) void {
    // Reap child process if it exits
    if (child_pid > 0) {
        var status: c_int = 0;
        const pid = posix.waitpid(child_pid, &status, posix.W.NOHANG);
        if (pid == child_pid) {
            log("Child {d} reaped with status {d}", .{ child_pid, status });
            child_pid = 0;
        }
    }
}

// =============================================================================
// VSock Communication (TCP fallback)
// =============================================================================

/// connectToHost establishes a connection to the host.
/// Uses COCO_VSOCK_CID env var for hypervisor port forwarding selection.
/// Falls back to TCP connection since AF_VSOCK is not in Zig std.
fn connectToHost(port: u32) !std.net.Stream {
    // In a real vsock implementation, we would use:
    //   const addr = try std.net.Address.initVsock(cid, port);
    // Since Zig 0.14.0 std doesn't include vsock support, we use TCP
    // with the expectation that the hypervisor maps vsock port to a TCP port.
    // The host listens on the same port number via vsock-to-TCP mapping.

    const loopback = try std.net.Address.parseIp("127.0.0.1", port);
    log("Connecting to 127.0.0.1:{d} (vsock fallback)", .{port});
    return try loopback.connect();
}

/// sendChunk sends a stream chunk to host with proper framing
fn sendChunk(sock: std.net.Stream, stream_type: u32, data: []const u8) !void {
    // Length-prefixed message: [4-byte stream_type][4-byte payload_len][payload]
    var header: [8]u8 = undefined;
    std.mem.writeIntLittle(u32, &header[0..4], stream_type);
    std.mem.writeIntLittle(u32, header[4..8], @as(u32, @intCast(data.len)));

    try sock.writeAll(&header);
    if (data.len > 0) {
        try sock.writeAll(data);
    }
}

/// sendExit sends the exit code to host
fn sendExit(sock: std.net.Stream, code: u32) !void {
    // Exit message: [stream_type=3][payload_len=4][exit_code]
    var header: [8]u8 = undefined;
    std.mem.writeIntLittle(u32, &header[0..4], STREAM_EXIT);
    std.mem.writeIntLittle(u32, header[4..8], 4);
    try sock.writeAll(&header);

    var exit_code: [4]u8 = undefined;
    std.mem.writeIntLittle(u32, &exit_code, code);
    try sock.writeAll(&exit_code);
}

/// recvMessage receives a length-prefixed message from vsock
fn recvMessage(sock: std.net.Stream, buf: []u8) ![]u8 {
    // Read 4-byte length header
    var len_buf: [4]u8 = undefined;
    try sock.readFull(&len_buf);
    const payload_len = std.mem.readIntLittle(u32, &len_buf);

    if (payload_len > buf.len) {
        return error.MessageTooLarge;
    }

    // Read payload
    try sock.readFull(buf[0..payload_len]);
    return buf[0..payload_len];
}

// =============================================================================
// Command Execution with Streaming
// =============================================================================

/// executeStreamingCommand executes a command and streams stdout/stderr back
fn executeStreamingCommand(sock: std.net.Stream, cmdline: []const u8) !void {
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
    var child = std.ChildProcess.init(argv, allocator);
    child.stdout_behavior = .pipe;
    child.stderr_behavior = .pipe;

    // Spawn
    child_pid = try child.spawn();
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
fn runMainLoop(sock: std.net.Stream) !void {
    log("Main loop started", .{});

    var recv_buf: [RECV_BUF_SIZE]u8 = undefined;

    while (running) {
        // Wait for socket to be readable or timeout (for checking running flag)
        sock.setReadTimeout(100 * std.time.ns_per_ms) catch {};

        const msg = recvMessage(sock, &recv_buf) catch |e| {
            switch (e) {
                error.WouldBlock => {
                    // Timeout, check running flag
                    continue;
                },
                error.ConnectionReset, error.ConnectionAborted => {
                    log("Connection closed by host", .{});
                    break;
                },
                else => {
                    logError("Receive error: {}", .{e});
                    break;
                },
            }
        };

        if (msg.len < 4) {
            logError("Invalid message length: {d}", .{msg.len});
            continue;
        }

        // Parse message type (first 4 bytes)
        const msg_type = std.mem.readIntLittle(u32, msg[0..4]);
        const payload = msg[4..];

        switch (msg_type) {
            CMD_EXEC => {
                // payload is the command line string
                const cmdline = payload;
                if (cmdline.len == 0 or cmdline[cmdline.len - 1] == 0) {
                    // Remove trailing null if present
                    executeStreamingCommand(sock, cmdline[0 .. cmdline.len - 1]) catch |e| {
                        logError("Execute failed: {}", .{e});
                    };
                } else {
                    executeStreamingCommand(sock, cmdline) catch |e| {
                        logError("Execute failed: {}", .{e});
                    };
                }
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

/// setupPid1 sets up PID 1 specific requirements
fn setupPid1() !void {
    // Create essential directories if they don't exist
    try ensureDir("/proc");
    try ensureDir("/sys");
    try ensureDir("/dev");
    try ensureDir("/tmp");
    try ensureDir("/var/log");
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
    allocator = gpa.allocator();

    // Parse environment for vsock port (CID is not used in TCP fallback)
    var vsock_port_val: u32 = DEFAULT_VSOCK_PORT;
    if (std.posix.getenv("COCO_VSOCK_PORT")) |port_str| {
        vsock_port_val = std.fmt.parseInt(u32, std.mem.sliceTo(port_str, 0), 10) catch DEFAULT_VSOCK_PORT;
    }

    log("Guest agent starting (PID 1)", .{});
    log("VSock port: {d} (TCP fallback)", .{vsock_port_val});

    // Setup PID 1
    try setupPid1();

    // Setup signal handlers
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
