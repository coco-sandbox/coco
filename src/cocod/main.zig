// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cocod — Guest agent running inside MicroVM as PID 1.
//! Listens on vsock for exec commands from the host.
//! Streams stdout/stderr back via vsock connection.

const std = @import("std");

// =============================================================================
// Constants
// =============================================================================

const VSOCK_HOST_CID: u32 = 2;
const DEFAULT_VSOCK_PORT: u32 = 4747;
const RECV_BUF_SIZE: usize = 65536;
const SEND_BUF_SIZE: usize = 65536;
const LINE_BUF_SIZE: usize = 4096;

// Stream types for vsock protocol
const STREAM_STDOUT: u32 = 1;
const STREAM_STDERR: u32 = 2;
const STREAM_EXIT: u32 = 3;

// =============================================================================
// Protocol
// =============================================================================

// Request: exec <cmdline>\n
// Response: stdout <data>\n | stderr <data>\n | exit <code>\n

const VsockMessage = struct {
    stream_type: u32,
    payload_len: u32,
};

const ExecRequest = struct {
    cmd_len: u32,
    args_len: u32,
    env_len: u32,
    working_dir_len: u32,
};

// =============================================================================
// Global State
// =============================================================================

var vsock_port: u32 = DEFAULT_VSOCK_PORT;
var verbose: bool = false;

// =============================================================================
// Logging
// =============================================================================

fn log(comptime fmt: []const u8, args: anytype) void {
    if (verbose) {
        std.debug.print("[cocod] " ++ fmt ++ "\n", args);
    }
}

fn logError(comptime fmt: []const u8, args: anytype) void {
    std.debug.print("[cocod] ERROR: " ++ fmt ++ "\n", args);
}

// =============================================================================
// VSock Communication
// =============================================================================

/// connectToHost connects to the host via vsock.
/// In real implementation, this uses AF_VSOCK with CID 2 (host).
/// For now, uses Unix socket as fallback in non-VM environments.
fn connectToHost(port: u32) !std.net.Stream {
    // In production, we'd use:
    // const addr = try std.net.Address.initVsock(2, port);
    // return try addr.connect();

    // Fallback to Unix socket for testing outside VM
    const sock_path = try std.fmt.allocPrint(std.heap.page_allocator, "/tmp/cocod.sock.{d}", .{port});
    log("Connecting to {s}", .{sock_path});
    return try std.net.connectUnixSocket(sock_path);
}

/// sendStream sends a stream chunk to host
fn sendStream(sock: std.net.Stream, stream_type: u32, data: []const u8) !void {
    var header: [12]u8 = undefined;
    std.mem.writeIntLittle(u32, &header, 102); // RESP_EXEC
    std.mem.writeIntLittle(u32, header[4..8], @as(u32, @intCast(8 + data.len)));
    std.mem.writeIntLittle(u32, header[8..12], stream_type);
    try sock.writeAll(&header);
    try sock.writeAll(data);
}

/// sendExit sends exit code to host
fn sendExit(sock: std.net.Stream, code: u32) !void {
    var header: [20]u8 = undefined;
    std.mem.writeIntLittle(u32, &header, 102); // RESP_EXEC
    std.mem.writeIntLittle(u32, header[4..8], 12);
    std.mem.writeIntLittle(u32, header[8..12], 3); // STREAM_EXIT
    std.mem.writeIntLittle(u32, header[12..16], code);
    std.mem.writeIntLittle(u32, header[16..20], 0);
    try sock.writeAll(&header);
}

// =============================================================================
// Command Execution
// =============================================================================

/// executeCommand parses and executes a command from the host
fn executeCommand(sock: std.net.Stream, cmdline: []const u8) !void {
    log("Executing: {s}", .{cmdline});

    // Parse command line into argv
    var args = std.ArrayList([]const u8).init(std.heap.page_allocator);
    var it = std.mem.splitSequence(u8, cmdline, " ");
    while (it.next()) |arg| {
        if (arg.len > 0) {
            try args.append(arg);
        }
    }

    if (args.items.len == 0) {
        try sendStream(sock, STREAM_STDERR, "No command provided\n");
        try sendExit(sock, 1);
        return;
    }

    const argv = try args.toOwnedSlice();
    defer std.heap.page_allocator.free(argv);

    // Execute
    const result = try std.process.exec(.{
        .allocator = std.heap.page_allocator,
        .argv = argv,
    });

    // Send stdout
    if (result.stdout.len > 0) {
        try sendStream(sock, STREAM_STDOUT, result.stdout);
    }

    // Send stderr
    if (result.stderr.len > 0) {
        try sendStream(sock, STREAM_STDERR, result.stderr);
    }

    // Send exit code
    try sendExit(sock, @as(u32, @intCast(result.term.Exited)));

    // Free result
    if (result.stdout.len > 0) std.heap.page_allocator.free(result.stdout);
    if (result.stderr.len > 0) std.heap.page_allocator.free(result.stderr);
}

// =============================================================================
// Main Loop
// =============================================================================

/// runMainLoop listens for commands from host and executes them
fn runMainLoop(sock: std.net.Stream) !void {
    log("Main loop started", .{});

    var buf: [RECV_BUF_SIZE]u8 = undefined;
    var offset: usize = 0;

    while (true) {
        const n = sock.read(buf[offset..]) catch {
            logError("Socket read error", .{});
            break;
        };

        if (n == 0) {
            log("Connection closed", .{});
            break;
        }

        offset += n;

        // Process complete lines
        var newline_idx: ?usize = null;
        for (buf[0..offset], 0..) |byte, i| {
            if (byte == '\n') {
                newline_idx = i;
                break;
            }
        }

        if (newline_idx) |idx| {
            const line = buf[0..idx];
            const remaining = offset - idx - 1;

            // Handle exec command
            if (std.mem.startsWith(u8, line, "exec ")) {
                const cmdline = line[4..];
                executeCommand(sock, cmdline) catch |e| {
                    logError("Execute failed: {}", .{e});
                };
            } else if (std.mem.startsWith(u8, line, "ping")) {
                try sendStream(sock, STREAM_STDOUT, "pong\n");
            } else if (std.mem.startsWith(u8, line, "shutdown")) {
                log("Shutdown requested", .{});
                break;
            } else {
                log("Unknown command: {s}", .{line});
            }

            // Shift remaining
            if (remaining > 0) {
                std.mem.copyForwards(u8, buf[0..remaining], buf[idx + 1 .. offset]);
            }
            offset = remaining;
        }
    }

    log("Main loop ended", .{});
}

// =============================================================================
// Signal Handling
// =============================================================================

/// setupSignalHandlers sets up graceful shutdown on SIGTERM
fn setupSignalHandlers() void {
    // In real implementation, we'd register SIGTERM handler
    // to gracefully shut down the main loop
    log("Signal handlers registered (placeholder)", .{});
}

// =============================================================================
// Main Entry Point
// =============================================================================

pub fn main() !void {
    // Parse arguments
    var args = std.process.args();
    _ = args.next(); // skip program name

    while (args.next()) |arg| {
        if (std.mem.eql(u8, arg, "-v")) {
            verbose = true;
        } else if (std.mem.eql(u8, arg, "-p")) {
            if (args.next()) |port_str| {
                vsock_port = try std.fmt.parseInt(u32, port_str, 10);
            }
        } else if (std.mem.eql(u8, arg, "-h")) {
            std.debug.print("Usage: cocod [-v] [-p <port>]\n", .{});
            std.debug.print("  -v  Verbose logging\n", .{});
            std.debug.print("  -p  VSock port (default: 4747)\n", .{});
            return;
        }
    }

    std.debug.print("[cocod] Guest agent starting (PID 1)\n", .{});
    std.debug.print("[cocod] VSock port: {d}\n", .{vsock_port});

    setupSignalHandlers();

    // Ensure /proc, /sys, /dev exist (required for Linux namespaces)
    std.fs.makeDirAbsolute("/proc") catch {};
    std.fs.makeDirAbsolute("/sys") catch {};
    std.fs.makeDirAbsolute("/dev") catch {};
    std.fs.makeDirAbsolute("/tmp") catch {};
    std.fs.makeDirAbsolute("/var/log") catch {};

    // Try to connect to host
    var connected = false;
    var retries: u32 = 0;
    const max_retries: u32 = 10;

    while (!connected and retries < max_retries) {
        if (connectToHost(vsock_port)) |sock| {
            connected = true;
            std.debug.print("[cocod] Connected to host\n", .{});
            runMainLoop(sock) catch |e| {
                std.debug.print("[cocod] Loop error: {}\n", .{e});
            };
            sock.close();
        } else {
            retries += 1;
            std.debug.print("[cocod] Waiting for host (attempt {d}/{d})...\n", .{
                retries, max_retries,
            });
            std.time.sleep(1 * NS_PER_SEC);
        }
    }

    if (!connected) {
        std.debug.print("[cocod] Could not connect to host, exiting\n", .{});
        return error.HostUnreachable;
    }

    std.debug.print("[cocod] Shutdown complete\n", .{});
}

const NS_PER_SEC: u64 = 1_000_000_000;
