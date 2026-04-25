// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cocovisor — Hypervisor wrapper daemon.
//! Listens on /run/coco/visor.sock for boot/exec/destroy/pause/resume requests.

const std = @import("std");
const vmm = @import("vmm");

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

    // Extract variable-length string fields in protocol order
    const sandbox_id = try std.fmt.allocPrint(
        std.heap.page_allocator,
        "{s}",
        .{payload[base..][0..req.sandbox_id_len]},
    );
    const rootfs = try std.fmt.allocPrint(
        std.heap.page_allocator,
        "{s}",
        .{payload[base + req.sandbox_id_len ..][0..req.rootfs_path_len]},
    );
    const kernel = try std.fmt.allocPrint(
        std.heap.page_allocator,
        "{s}",
        .{payload[base + req.sandbox_id_len + req.rootfs_path_len ..][0..req.kernel_path_len]},
    );
    const initrd = if (req.initrd_path_len > 0)
        try std.fmt.allocPrint(
            std.heap.page_allocator,
            "{s}",
            .{payload[base + req.sandbox_id_len + req.rootfs_path_len + req.kernel_path_len ..][0..req.initrd_path_len]},
        )
    else
        "";

    const vsock_cid = next_vsock_cid;
    next_vsock_cid +%= 1;
    sandbox_id_counter += 1;

    const config: vmm.VMConfig = .{
        .id = sandbox_id,
        .rootfs = rootfs,
        .kernel = kernel,
        .initrd = initrd,
        .memory_mb = req.memory_mb,
        .vcpus = req.vcpu_count,
        .vsock_cid = vsock_cid,
    };

    var vm = std.heap.page_allocator.create(vmm.VM) catch return;
    vm.* = vmm.VM.init(config);
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
    const args = payload[base + req.cmd_len ..][0..req.args_len];
    _ = payload[base + req.cmd_len + req.args_len ..][0..req.env_len]; // env (unused in stub)
    _ = payload[base + req.cmd_len + req.args_len + req.env_len ..][0..req.working_dir_len]; // working_dir (unused in stub)

    std.debug.print("[cocovisor] Exec: {s} args={s}\n", .{ cmd, args });

    // TODO: Use posix.fork() + posix.execvpeZ() for real exec
    // For skeleton, return mock output
    const mock_output = "mock exec output";

    sendExecChunk(sock, 1, mock_output, 0) catch {};
    sendExecChunk(sock, 3, "", 0) catch {}; // exit chunk
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
        vm.hibernate() catch |e| {
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
        vm.resumeFromHibernate() catch |e| {
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

// =============================================================================
// Main
// =============================================================================

pub fn main() !void {
    std.debug.print("[cocovisor] Starting daemon on {s}\n", .{SOCK_PATH});

    std.fs.deleteFileAbsolute(SOCK_PATH) catch {};
    std.fs.makeDirAbsolute("/run/coco") catch {};
    std.fs.makeDirAbsolute("/var/lib/coco/hibernation") catch {};

    const sock_addr = try std.net.Address.initUnix(SOCK_PATH);
    var listener = try sock_addr.listen(.{ .reuse_address = true });

    std.debug.print("[cocovisor] Listening on {s}\n", .{SOCK_PATH});
    std.debug.print("[cocovisor] Protocol: BOOT={d}, EXEC={d}, DESTROY={d}, PAUSE={d}, RESUME={d}, GET_STATE={d}, FORK={d}, HIBERNATE={d}\n", .{
        REQ_BOOT, REQ_EXEC, REQ_DESTROY, REQ_PAUSE, REQ_RESUME, REQ_GET_STATE, REQ_FORK, REQ_HIBERNATE,
    });

    while (true) {
        const conn = try listener.accept();
        const t = std.Thread.spawn(.{}, handleConnection, .{conn.stream}) catch continue;
        t.detach();
    }
}
