// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cocovisor — Hypervisor wrapper daemon.
//! Listens on /run/coco/visor.sock for boot/exec/destroy requests.

const std = @import("std");

// =============================================================================
// Protocol Constants
// =============================================================================

const REQ_BOOT: u32 = 1;
const REQ_EXEC: u32 = 2;
const REQ_DESTROY: u32 = 3;
const REQ_GET_STATE: u32 = 6;

const RESP_BOOT: u32 = 101;
const RESP_EXEC: u32 = 102;
const RESP_DESTROY: u32 = 103;
const RESP_GET_STATE: u32 = 106;
const RESP_ERROR: u32 = 199;

const SOCK_PATH = "/run/coco/visor.sock";

// =============================================================================
// Binary Protocol Frame
// =============================================================================

const Frame = struct {
    kind: u32,
    size: u32,

    fn init(kind: u32, size: u32) Frame {
        return .{ .kind = kind, .size = size };
    }

    fn headerSize() usize {
        return @sizeOf(u32) * 2;
    }
};

// =============================================================================
// Visor Protocol Types
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

const ExecRequest = struct {
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

const GetStateResponse = struct {
    state: u32,
    pid: u32,
    vsock_cid: u32,
};

// =============================================================================
// Sandbox State
// =============================================================================

const SandboxState = enum(u32) {
    created = 0,
    booting = 1,
    running = 2,
    paused = 3,
    hibernated = 4,
    stopping = 5,
    stopped = 6,
    err_state = 7,
};

const Sandbox = struct {
    id: []const u8,
    state: SandboxState,
    pid: u32,
    vsock_cid: u32,
    rootfs: []const u8,
    memory_mb: u32,
    vcpus: u32,
};

// =============================================================================
// Global State
// =============================================================================

var sandboxes = std.StringHashMap(Sandbox);
var next_vsock_cid: u32 = 3;

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

        // Try to process complete frames
        while (offset >= Frame.headerSize()) {
            const kind = std.mem.readIntLittle(u32, buf[0..4]);
            const size = std.mem.readIntLittle(u32, buf[4..8]);

            if (offset < Frame.headerSize() + size) break; // incomplete

            const payload = buf[Frame.headerSize() .. Frame.headerSize() + size];
            handleFrame(sock, kind, payload) catch return;
            offset -= Frame.headerSize() + size;

            // Shift remaining data
            if (offset > 0) {
                std.mem.copyForwards(u8, buf[0..offset], buf[Frame.headerSize() + size .. offset + Frame.headerSize() + size]);
            }
        }
    }
}

fn handleFrame(sock: std.net.Stream, kind: u32, payload: []u8) !void {
    switch (kind) {
        REQ_BOOT => try handleBoot(sock, payload),
        REQ_EXEC => try handleExec(sock, payload),
        REQ_DESTROY => try handleDestroy(sock, payload),
        REQ_GET_STATE => try handleGetState(sock, payload),
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

    const sandbox_id = payload[@sizeOf(BootRequest) .. @sizeOf(BootRequest) + req.sandbox_id_len];
    const rootfs_path = payload[@sizeOf(BootRequest) + req.sandbox_id_len ..][0..req.rootfs_path_len];

    const id = try std.fmt.allocPrint(std.heap.page_allocator, "{s}", .{sandbox_id});
    const rootfs = try std.fmt.allocPrint(std.heap.page_allocator, "{s}", .{rootfs_path});

    const vsock_cid = next_vsock_cid;
    next_vsock_cid += 1;

    const sandbox = Sandbox{
        .id = id,
        .state = .running,
        .pid = 12345 + vsock_cid, // placeholder
        .vsock_cid = vsock_cid,
        .rootfs = rootfs,
        .memory_mb = req.memory_mb,
        .vcpus = req.vcpu_count,
    };

    try sandboxes.put(id, sandbox);

    std.debug.print("[cocovisor] Boot sandbox {s} (cid={d}, pid={d})\n", .{ id, vsock_cid, sandbox.pid });

    // Send response
    var resp: [16]u8 = undefined;
    std.mem.writeIntLittle(u32, &resp, RESP_BOOT);
    std.mem.writeIntLittle(u32, resp[4..8], 12);
    std.mem.writeIntLittle(u32, resp[8..12], vsock_cid);
    std.mem.writeIntLittle(u32, resp[12..16], sandbox.pid);
    try sock.writeAll(&resp);
}

fn handleExec(sock: std.net.Stream, payload: []u8) !void {
    if (payload.len < @sizeOf(ExecRequest)) {
        try sendError(sock, "Exec request too small");
        return;
    }

    const req = @as(*align(1) const ExecRequest, @ptrCast(payload.ptr));
    const cmd = payload[@sizeOf(ExecRequest) .. @sizeOf(ExecRequest) + req.cmd_len];

    std.debug.print("[cocovisor] Exec: {s}\n", .{cmd});

    // Send stdout chunk
    var stdout_hdr: [12]u8 = undefined;
    std.mem.writeIntLittle(u32, &stdout_hdr, RESP_EXEC);
    std.mem.writeIntLittle(u32, stdout_hdr[4..8], 8 + 26);
    std.mem.writeIntLittle(u32, stdout_hdr[8..12], 1); // stdout
    const data_offset: u32 = 26;
    std.mem.writeIntLittle(u32, stdout_hdr[8..12], data_offset);

    var chunk: [38]u8 = undefined;
    std.mem.copySlice(chunk[0..12], stdout_hdr[0..12]);
    std.mem.writeIntLittle(u32, chunk[12..16], 26); // data_len
    std.mem.writeIntLittle(u32, chunk[16..20], 0); // exit_code
    const msg = "Hello from sandbox exec!\n";
    std.mem.copySlice(chunk[20 .. 20 + 26], msg);

    try sock.writeAll(&chunk);

    // Send exit chunk
    var exit_chunk: [20]u8 = undefined;
    std.mem.writeIntLittle(u32, &exit_chunk, RESP_EXEC);
    std.mem.writeIntLittle(u32, exit_chunk[4..8], 8);
    std.mem.writeIntLittle(u32, exit_chunk[8..12], 3); // exit
    std.mem.writeIntLittle(u32, exit_chunk[12..16], 0); // exit_code
    std.mem.writeIntLittle(u32, exit_chunk[16..20], 0);
    try sock.writeAll(&exit_chunk);
}

fn handleDestroy(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];

    std.debug.print("[cocovisor] Destroy: {s}\n", .{id});

    _ = sandboxes.remove(id);

    var resp: [12]u8 = undefined;
    std.mem.writeIntLittle(u32, &resp, RESP_DESTROY);
    std.mem.writeIntLittle(u32, resp[4..8], 0);
    std.mem.writeIntLittle(u32, resp[8..12], 0);
    try sock.writeAll(&resp);
}

fn handleGetState(sock: std.net.Stream, payload: []u8) !void {
    const id = payload[0..payload.len];

    if (sandboxes.get(id)) |sb| {
        var resp: [16]u8 = undefined;
        std.mem.writeIntLittle(u32, &resp, RESP_GET_STATE);
        std.mem.writeIntLittle(u32, resp[4..8], 12);
        std.mem.writeIntLittle(u32, resp[8..12], @intFromEnum(sb.state));
        std.mem.writeIntLittle(u32, resp[12..16], sb.pid);
        try sock.writeAll(&resp);
    } else {
        var resp: [12]u8 = undefined;
        std.mem.writeIntLittle(u32, &resp, RESP_ERROR);
        std.mem.writeIntLittle(u32, resp[4..8], 6);
        const msg = "NOTFND";
        std.mem.copySlice(resp[8..14], msg);
        try sock.writeAll(&resp);
    }
}

fn sendError(sock: std.net.Stream, msg: []const u8) !void {
    var resp: [8]u8 = undefined;
    std.mem.writeIntLittle(u32, &resp, RESP_ERROR);
    std.mem.writeIntLittle(u32, resp[4..8], @as(u32, @intCast(msg.len)));
    try sock.writeAll(&resp);
    try sock.writeAll(msg);
}

// =============================================================================
// Main
// =============================================================================

pub fn main() !void {
    std.debug.print("[cocovisor] Starting daemon on {s}\n", .{SOCK_PATH});

    // Clean up old socket
    std.fs.deleteFileAbsolute(SOCK_PATH) catch {};

    // Ensure directory exists
    std.fs.makeDirAbsolute("/run/coco") catch {};

    const sock_addr = try std.net.Address.initUnix(SOCK_PATH);
    const listener = try sock_addr.listen(.{ .reuse_address = true });

    std.debug.print("[cocovisor] Listening on {s}\n", .{SOCK_PATH});

    while (true) {
        const conn = try listener.accept();
        const t = std.Thread.spawn(.{}, handleConnection, .{conn}) catch continue;
        t.detach();
    }
}
