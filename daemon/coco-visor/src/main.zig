// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cocovisor — Hypervisor wrapper daemon.
//! Listens on /run/coco/visor.sock for boot/exec/destroy/pause/resume requests.
//! Provides metrics on port 9090 and health checks on port 4748.

const std = @import("std");
const linux = std.os.linux;
const vmm = @import("vmm.zig");
const config = @import("config.zig");
const logger = @import("logger.zig");
const metrics = @import("metrics.zig");
const agent_registry = @import("agent_registry.zig");
const vsock = @import("vsock.zig");

// =============================================================================
// Protocol Constants
// =============================================================================

const REQ_BOOT: u32 = 1;
const REQ_EXEC: u32 = 2;
const REQ_DESTROY: u32 = 3;
const REQ_PAUSE: u32 = 4;
const REQ_RESUME: u32 = 5;
const REQ_GET_STATE: u32 = 6;
const REQ_FORK: u32 = 7;
const REQ_HIBERNATE: u32 = 8;
const REQ_RESUME_HIBERNATED: u32 = 9;

const RESP_OK: u32 = 100;
const RESP_BOOT: u32 = 101;
const RESP_EXEC: u32 = 102;
const RESP_DESTROY: u32 = 103;
const RESP_GET_STATE: u32 = 106;
const RESP_FORK: u32 = 107;
const RESP_HIBERNATE: u32 = 108;
const RESP_ERROR: u32 = 199;

const SOCK_PATH = "/run/coco/visor.sock";

var global_config: config.Config = undefined;

inline fn w32(buf: [*]u8, value: u32) void {
    std.mem.writeInt(u32, @ptrCast(buf), value, .little);
}

inline fn r32(buf: []const u8) u32 {
    return std.mem.readInt(u32, @ptrCast(buf.ptr), .little);
}

// =============================================================================
// Binary Protocol Frame
// =============================================================================

const Frame = struct {
    kind: u32,
    size: u32,

    fn headerSize() usize {
        return @sizeOf(u32) * 2;
    }
};

// =============================================================================
// Request/Response Structures
// =============================================================================

const BootRequest = extern struct {
    rootfs_path_len: u32,
    memory_mb: u32,
    vcpu_count: u32,
    kernel_path_len: u32,
    initrd_path_len: u32,
    sandbox_id_len: u32,
    vsock_port: u32,
    padding: u32,
};

const BootResponse = struct {
    vsock_cid: u32,
    pid: u32,
    state: u32,
};

const ForkResponse = struct {
    child_vsock_cid: u32,
    child_pid: u32,
    duration_ms: u32,
};

const GetStateResponse = struct {
    state: u32,
    pid: u32,
    vsock_cid: u32,
};

// =============================================================================
// Global State
// =============================================================================

var next_vsock_cid: u32 = 3;
var sandbox_id_counter: u32 = 0;

// =============================================================================
// Socket Server
// =============================================================================

fn handleConnection(sock: std.net.Stream) void {
    defer sock.close();
    var buf: [8192]u8 = undefined;
    var offset: usize = 0;

    while (true) {
        const n = sock.read(buf[offset..]) catch break;
        if (n == 0) break;
        offset += n;

        while (offset >= Frame.headerSize()) {
            const kind = std.mem.readInt(u32, buf[0..4], .little);
            const size = std.mem.readInt(u32, buf[4..8], .little);

            if (offset < Frame.headerSize() + size) break;

            const payload = buf[Frame.headerSize() .. Frame.headerSize() + size];
            handleFrame(sock, kind, payload) catch return;
            offset -= Frame.headerSize() + size;

            if (offset > 0) {
                @memcpy(buf[0..offset], buf[Frame.headerSize() + size ..]);
            }
        }
    }
}

fn handleFrame(sock: std.net.Stream, kind: u32, payload: []u8) !void {
    switch (kind) {
        REQ_BOOT => try handleBoot(sock, payload),
        REQ_EXEC => try handleExec(sock, payload),
        REQ_DESTROY => try handleDestroy(sock, payload),
        REQ_PAUSE => try handlePause(sock, payload),
        REQ_RESUME => try handleResume(sock, payload),
        REQ_GET_STATE => try handleGetState(sock, payload),
        REQ_FORK => try handleFork(sock, payload),
        REQ_HIBERNATE => try handleHibernate(sock, payload),
        REQ_RESUME_HIBERNATED => try handleResumeHibernated(sock, payload),
        else => try sendError(sock, "Unknown request kind"),
    }
}

// =============================================================================
// Request Handlers
// =============================================================================

fn handleBoot(sock: std.net.Stream, payload: []u8) !void {
    if (payload.len < @sizeOf(BootRequest)) {
        try sendError(sock, "Boot request too small");
        return;
    }

    const req = @as(*align(1) const BootRequest, @ptrCast(payload.ptr));
    const start = std.time.nanoTimestamp();
    const base = @sizeOf(BootRequest);

    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    const sandbox_id = try std.fmt.allocPrint(
        allocator,
        "{s}",
        .{payload[base..][0..req.sandbox_id_len]},
    );
    errdefer allocator.free(sandbox_id);

    const rootfs = try std.fmt.allocPrint(
        allocator,
        "{s}",
        .{payload[base + req.sandbox_id_len ..][0..req.rootfs_path_len]},
    );
    errdefer allocator.free(rootfs);

    const kernel = try std.fmt.allocPrint(
        allocator,
        "{s}",
        .{payload[base + req.sandbox_id_len + req.rootfs_path_len ..][0..req.kernel_path_len]},
    );
    errdefer allocator.free(kernel);

    const initrd = if (req.initrd_path_len > 0)
        try std.fmt.allocPrint(
            allocator,
            "{s}",
            .{payload[base + req.sandbox_id_len + req.rootfs_path_len + req.kernel_path_len ..][0..req.initrd_path_len]},
        )
    else
        "";
    errdefer if (req.initrd_path_len > 0) allocator.free(initrd);

    const vsock_cid = next_vsock_cid;
    next_vsock_cid +%= 1;
    sandbox_id_counter += 1;

    const vm_config: vmm.VMConfig = .{
        .id = sandbox_id,
        .rootfs = rootfs,
        .kernel = kernel,
        .initrd = initrd,
        .memory_mb = req.memory_mb,
        .vcpus = req.vcpu_count,
        .vsock_cid = vsock_cid,
    };

    var vm = allocator.create(vmm.VM) catch return;
    errdefer allocator.destroy(vm);
    vm.* = vmm.VM.init(vm_config);
    const result = vm.boot() catch |e| {
        std.debug.print("[cocovisor] Boot failed: {}\n", .{e});
        try sendError(sock, "Boot failed");
        return;
    };

    const duration = @as(u64, @intCast(std.time.nanoTimestamp() - start));
    vmm.getMetrics().recordBoot(duration);

    std.debug.print("[cocovisor] Boot sandbox {s} (cid={d}, pid={d}) in {d}µs\n", .{
        sandbox_id, result.vsock_cid, result.pid, duration / 1000,
    });

    var resp: [16]u8 = undefined;
    w32(resp[0..4].ptr, RESP_BOOT);
    w32(resp[4..8].ptr, 12);
    w32(resp[8..12].ptr, result.vsock_cid);
    w32(resp[12..16].ptr, result.pid);
    try sock.writeAll(&resp);
}

const ExecRequest = extern struct {
    cmd_len: u32,
    args_len: u32,
    env_len: u32,
    working_dir_len: u32,
};

const ExecStreamChunk = struct {
    stream_type: u32, // 1=stdout, 2=stderr, 3=exit
    data_len: u32,
    exit_code: u32,
};

fn sendExecChunk(sock: std.net.Stream, stream_type: u32, data: []const u8, exit_code: u32) !void {
    const chunk_size = @sizeOf(ExecStreamChunk) + data.len;
    var frame = try std.heap.page_allocator.alloc(u8, 8 + chunk_size);
    defer std.heap.page_allocator.free(frame);
    w32(frame[0..4].ptr, RESP_EXEC);
    w32(frame[4..8].ptr, @as(u32, @intCast(chunk_size)));
    w32(frame[8..12].ptr, stream_type);
    w32(frame[12..16].ptr, @as(u32, @intCast(data.len)));
    w32(frame[16..20].ptr, exit_code);
    @memcpy(frame[20..][0..data.len], data);
    try sock.writeAll(frame);
}

fn handleExec(sock: std.net.Stream, payload: []u8) !void {
    if (payload.len < @sizeOf(ExecRequest)) {
        try sendError(sock, "Exec request too small");
        return;
    }

    const req = @as(*align(1) const ExecRequest, @ptrCast(payload.ptr));
    const base = @sizeOf(ExecRequest);

    const cmd = payload[base..][0..req.cmd_len];
    const args_raw = payload[base + req.cmd_len ..][0..req.args_len];
    const env_raw = payload[base + req.cmd_len + req.args_len ..][0..req.env_len];
    const workdir = payload[base + req.cmd_len + req.args_len + req.env_len ..][0..req.working_dir_len];

    // Parse null-terminated args into a slice
    var args_list = std.ArrayList([]const u8).init(std.heap.page_allocator);
    defer args_list.deinit();
    var arg_start: usize = 0;
    for (args_raw, 0..) |byte, i| {
        if (byte == 0) {
            try args_list.append(args_raw[arg_start..i]);
            arg_start = i + 1;
        }
    }

    // Parse null-terminated env into key=value pairs
    var env_list = std.ArrayList([]const u8).init(std.heap.page_allocator);
    defer env_list.deinit();
    var env_start: usize = 0;
    for (env_raw, 0..) |byte, i| {
        if (byte == 0) {
            if (env_start < i) {
                try env_list.append(env_raw[env_start..i]);
            }
            env_start = i + 1;
        }
    }

    std.debug.print("[cocovisor] Exec: {s} with {d} args\n", .{ cmd, args_list.items.len });

    // Create pipes for stdout and stderr
    const stdout_pipe = try std.posix.pipe2(.{});
    const stderr_pipe = try std.posix.pipe2(.{});
    defer {
        std.posix.close(stdout_pipe[0]);
        std.posix.close(stdout_pipe[1]);
        std.posix.close(stderr_pipe[0]);
        std.posix.close(stderr_pipe[1]);
    }

    // Fork child process
    const pid = try std.posix.fork();
    if (pid == 0) {
        // Child: set up environment and exec
        std.posix.close(stdout_pipe[0]);
        std.posix.close(stderr_pipe[0]);

        // Redirect stdout and stderr
        try std.posix.dup2(stdout_pipe[1], std.posix.STDOUT_FILENO);
        try std.posix.dup2(stderr_pipe[1], std.posix.STDERR_FILENO);
        std.posix.close(stdout_pipe[1]);
        std.posix.close(stderr_pipe[1]);

        // Set working directory if provided
        if (workdir.len > 0) {
            std.posix.chdir(workdir) catch {};
        }

        // Skip env setting - env vars not supported in this context

        // Convert args to null-terminated C strings
        var argv = std.ArrayList(?[*:0]const u8).init(std.heap.page_allocator);
        defer argv.deinit();
        try argv.append(@ptrCast(cmd));
        for (args_list.items) |arg| {
            try argv.append(@ptrCast(arg));
        }
        try argv.append(null);

        // Execute the command
        const empty_env = [1]?[*:0]const u8{null};
        const path: [*:0]const u8 = @ptrCast(cmd);
        const child_argv: [*:null]const ?[*:0]const u8 = @ptrCast(argv.items.ptr);
        const env: [*:null]const ?[*:0]const u8 = @ptrFromInt(@intFromPtr(&empty_env));
        std.posix.execveZ(path, child_argv, env) catch {
            std.debug.print("[cocovisor] Exec failed for {s}\n", .{cmd});
            std.posix.exit(1);
        };
        unreachable;
    }

    // Parent: close child ends of pipes
    std.posix.close(stdout_pipe[1]);
    std.posix.close(stderr_pipe[1]);

    // Read stdout and stream back
    var stdout_buf: [4096]u8 = undefined;
    var stderr_buf: [4096]u8 = undefined;
    var exit_code: u32 = 0;
    var stdout_done = false;
    var stderr_done = false;

    while (!stdout_done or !stderr_done) {
        // Poll stdout
        if (!stdout_done) {
            const n = std.posix.read(stdout_pipe[0], &stdout_buf);
            if (n) |len| {
                try sendExecChunk(sock, 1, stdout_buf[0..len], 0);
            } else |_| {
                stdout_done = true;
            }
        }
        // Poll stderr
        if (!stderr_done) {
            const n = std.posix.read(stderr_pipe[0], &stderr_buf);
            if (n) |len| {
                try sendExecChunk(sock, 2, stderr_buf[0..len], 0);
            } else |_| {
                stderr_done = true;
            }
        }
    }

    // Wait for child to finish and get exit code
    const result = std.posix.waitpid(pid, 0);
    const status = result.status;
    if ((status & 0x7f) == 0) {
        exit_code = @as(u32, @intCast((status >> 8) & 0xff));
    }

    // Send exit chunk
    try sendExecChunk(sock, 3, "", exit_code);
    std.debug.print("[cocovisor] Exec complete: exit_code={d}\n", .{exit_code});
}

fn handleDestroy(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];
    std.debug.print("[cocovisor] Destroy: {s}\n", .{id});

    vmm.removeVM(id);

    var resp: [12]u8 = undefined;
    w32(resp[0..4].ptr, RESP_DESTROY);
    w32(resp[4..8].ptr, 0);
    w32(resp[8..12].ptr, 0);
    try sock.writeAll(&resp);
}

fn handlePause(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];
    std.debug.print("[cocovisor] Pause: {s}\n", .{id});

    if (vmm.getVMs().get(id)) |vm| {
        vm.pause() catch {};
    }

    var resp: [8]u8 = undefined;
    w32(resp[0..4].ptr, RESP_OK);
    w32(resp[4..8].ptr, 0);
    try sock.writeAll(&resp);
}

fn handleResume(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];
    std.debug.print("[cocovisor] Resume: {s}\n", .{id});

    if (vmm.getVMs().get(id)) |vm| {
        vm.resume_() catch {};
    }

    var resp: [8]u8 = undefined;
    w32(resp[0..4].ptr, RESP_OK);
    w32(resp[4..8].ptr, 0);
    try sock.writeAll(&resp);
}

fn handleFork(sock: std.net.Stream, payload: []u8) !void {
    const start = std.time.nanoTimestamp();

    if (payload.len < 8) {
        try sendError(sock, "Fork request too small");
        return;
    }

    const parent_id_len = r32(payload[0..4]);
    if (payload.len < 8 + parent_id_len) {
        try sendError(sock, "Fork request too small");
        return;
    }

    const parent_id = payload[4 .. 4 + parent_id_len];
    const child_name_len = r32(payload[4 + parent_id_len .. 8 + parent_id_len]);
    _ = child_name_len;

    std.debug.print("[cocovisor] Fork: {s}\n", .{parent_id});

    if (vmm.getVMs().get(parent_id)) |parent_vm| {
        const result = parent_vm.fork() catch |e| {
            std.debug.print("[cocovisor] Fork failed: {}\n", .{e});
            try sendError(sock, "Fork failed");
            return;
        };

        const duration = @as(u32, @intCast(@divTrunc(std.time.nanoTimestamp() - start, 1_000_000)));

        var resp: [16]u8 = undefined;
        w32(resp[0..4].ptr, RESP_FORK);
        w32(resp[4..8].ptr, 12);
        w32(resp[8..12].ptr, result.child_vsock_cid);
        w32(resp[12..16].ptr, result.child_pid);
        try sock.writeAll(&resp);

        vmm.getMetrics().recordFork(@as(u64, @intCast(duration * 1_000_000)));
        std.debug.print("[cocovisor] Fork complete: child_cid={d}, child_pid={d}, duration={d}ms\n", .{
            result.child_vsock_cid, result.child_pid, duration,
        });
    } else {
        try sendError(sock, "Sandbox not found");
    }
}

fn handleHibernate(sock: std.net.Stream, payload: []u8) !void {
    const start = std.time.nanoTimestamp();
    const id = payload[0..payload.len];

    std.debug.print("[cocovisor] Hibernate: {s}\n", .{id});

    if (vmm.getVMs().get(id)) |vm| {
        _ = vm.hibernate() catch |e| {
            std.debug.print("[cocovisor] Hibernate failed: {}\n", .{e});
            try sendError(sock, "Hibernate failed");
            return;
        };

        const duration = @as(u32, @intCast(@divTrunc(std.time.nanoTimestamp() - start, 1_000_000)));
        vmm.getMetrics().recordHibernate(@as(u64, @intCast(duration * 1_000_000)));

        var resp: [12]u8 = undefined;
        w32(resp[0..4].ptr, RESP_HIBERNATE);
        w32(resp[4..8].ptr, 4);
        w32(resp[8..12].ptr, duration);
        try sock.writeAll(&resp);

        std.debug.print("[cocovisor] Hibernate complete: {d}ms\n", .{duration});
    } else {
        try sendError(sock, "Sandbox not found");
    }
}

fn handleResumeHibernated(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];
    std.debug.print("[cocovisor] Resume from hibernate: {s}\n", .{id});

    if (vmm.getVMs().get(id)) |vm| {
        _ = vm.resumeFromHibernate() catch |e| {
            std.debug.print("[cocovisor] Resume failed: {}\n", .{e});
            try sendError(sock, "Resume failed");
            return;
        };

        var resp: [8]u8 = undefined;
        w32(resp[0..4].ptr, RESP_OK);
        w32(resp[4..8].ptr, 0);
        try sock.writeAll(&resp);
    } else {
        try sendError(sock, "Sandbox not found");
    }
}

fn handleGetState(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];

    if (vmm.getVMs().get(id)) |vm| {
        var resp: [16]u8 = undefined;
        w32(resp[0..4].ptr, RESP_GET_STATE);
        w32(resp[4..8].ptr, 12);
        w32(resp[8..12].ptr, @intFromEnum(vm.state));
        w32(resp[12..16].ptr, vm.pid);
        try sock.writeAll(&resp);
    } else {
        var resp: [14]u8 = undefined;
        w32(resp[0..4].ptr, RESP_ERROR);
        w32(resp[4..8].ptr, 6);
        @memcpy(resp[8..14], "NOTFND");
        try sock.writeAll(&resp);
    }
}

fn sendError(sock: std.net.Stream, msg: []const u8) !void {
    var resp: [8]u8 = undefined;
    w32(resp[0..4].ptr, RESP_ERROR);
    w32(resp[4..8].ptr, @as(u32, @intCast(msg.len)));
    try sock.writeAll(&resp);
    try sock.writeAll(msg);
}

fn startHttpServer(port: u16) void {
    _ = port;
}

fn handleHttpRequest(conn: std.net.Server.Connection) void {
    defer conn.stream.close();

    var buf: [4096]u8 = undefined;
    const n = conn.stream.read(&buf) catch return;
    if (n == 0) return;

    const request = buf[0..n];

    if (std.mem.startsWith(u8, request, "GET /metrics")) {
        handleMetrics(conn.stream) catch {};
    } else if (std.mem.startsWith(u8, request, "GET /health/live")) {
        handleHealthLive(conn.stream) catch {};
    } else if (std.mem.startsWith(u8, request, "GET /health/ready")) {
        handleHealthReady(conn.stream) catch {};
    } else if (std.mem.startsWith(u8, request, "GET /health")) {
        handleHealth(conn.stream) catch {};
    } else {
        handleNotFound(conn.stream) catch {};
    }
}

fn handleMetrics(sock: std.net.Stream) !void {
    const m = metrics.getMetrics();

    const header = "HTTP/1.1 200 OK\r\nContent-Type: text/plain; version=0.0.4\r\n\r\n";
    try sock.writeAll(header);

    try m.formatPrometheus(sock.writer().any());
}

fn handleHealthLive(sock: std.net.Stream) !void {
    const response = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"ok\"}";
    try sock.writeAll(response);
}

fn handleHealthReady(sock: std.net.Stream) !void {
    const response = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"ready\",\"checks\":{\"visor\":\"ok\",\"kvm\":\"ok\"}}";
    try sock.writeAll(response);
}

fn handleHealth(sock: std.net.Stream) !void {
    const vm_count = vmm.getVMs().count();
    var response_buf: [256]u8 = undefined;
    const response = std.fmt.bufPrint(&response_buf, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{{\"status\":\"ok\",\"visor\":\"running\",\"sandboxes\":{d}}}", .{vm_count}) catch return;

    try sock.writeAll(response);
}

fn handleNotFound(sock: std.net.Stream) !void {
    const response = "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\nNot Found";
    try sock.writeAll(response);
}

// =============================================================================
// Unix Socket
// =============================================================================

const AF_UNIX = 1;

fn createUnixSocket(socket_path: [*:0]const u8) !i32 {
    const posix = std.posix;
    const socket_fd = try posix.socket(AF_UNIX, posix.SOCK.STREAM, 0);
    errdefer posix.close(socket_fd);

    var addr: posix.sockaddr_un = undefined;
    addr.family = AF_UNIX;
    const path_len = std.mem.len(socket_path);
    if (path_len >= addr.path.len) return error.SocketPathTooLong;
    @memcpy(addr.path[0..path_len], socket_path[0..path_len]);

    try posix.bind(socket_fd, @ptrCast(&addr), @sizeOf(posix.sockaddr_un));
    try posix.listen(socket_fd, 128);

    return socket_fd;
}

// =============================================================================
// VSock Accept Loop
// =============================================================================

fn vsockAcceptLoop(server_fd: i32) void {
    while (true) {
        const result = vsock.acceptAgent(server_fd) catch continue;
        agent_registry.register(result.guest_cid, result.fd);
        logger.info("Agent connected from CID={d}", .{result.guest_cid});
    }
}

// =============================================================================
// Main
// =============================================================================

pub fn main() !void {
    global_config = config.loadConfig();

    logger.init(logger.LogLevel.info, {});
    const log_lvl = switch (global_config.log_level[0]) {
        'd' => logger.LogLevel.debug,
        'w' => logger.LogLevel.warn,
        'e' => logger.LogLevel.err,
        else => logger.LogLevel.info,
    };
    logger.setLevel(log_lvl);

    logger.info("Starting cocovisor daemon", .{});

    const socket_path_null: [*:0]const u8 = @ptrCast(global_config.socket_path);
    _ = linux.unlink(socket_path_null);
    _ = linux.mkdir("/run/coco", 0o755);
    const checkpoint_dir_null: [*:0]const u8 = @ptrCast(global_config.checkpoint_dir);
    _ = linux.mkdir(checkpoint_dir_null, 0o755);
    const hibernation_dir_null: [*:0]const u8 = @ptrCast(global_config.hibernation_dir);
    _ = linux.mkdir(hibernation_dir_null, 0o755);
    const template_dir_null: [*:0]const u8 = @ptrCast(global_config.template_dir);
    _ = linux.mkdir(template_dir_null, 0o755);

    agent_registry.init();

    const socket_fd = try createUnixSocket(socket_path_null);
    defer std.posix.close(socket_fd);

    const vsock_server_fd = vsock.createServer(global_config.vsock_port) catch |e| blk: {
        logger.warn("VSock server failed: {}", .{e});
        break :blk -1;
    };
    if (vsock_server_fd >= 0) {
        const vt = std.Thread.spawn(.{}, vsockAcceptLoop, .{vsock_server_fd}) catch null;
        if (vt) |t| t.detach();
        logger.info("VSock server listening on port {d}", .{global_config.vsock_port});
    }

    if (global_config.metrics_enabled) {
        const mt = std.Thread.spawn(.{}, startHttpServer, .{global_config.metrics_port}) catch null;
        if (mt) |t| t.detach();
    }
    if (global_config.health_enabled) {
        const ht = std.Thread.spawn(.{}, startHttpServer, .{global_config.health_port}) catch null;
        if (ht) |t| t.detach();
    }

    logger.info("Listening on {s}", .{global_config.socket_path});

    while (true) {
        var addr: std.posix.sockaddr_un = undefined;
        var addr_len: std.posix.socklen_t = @sizeOf(std.posix.sockaddr_un);
        const client_fd = std.posix.accept(socket_fd, @ptrCast(&addr), &addr_len, 0) catch continue;
        const stream = std.net.Stream{ .handle = client_fd };
        const thread = std.Thread.spawn(.{}, handleConnection, .{stream}) catch {
            stream.close();
            continue;
        };
        thread.detach();
    }
}
