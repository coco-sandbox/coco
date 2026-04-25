// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const net = std.net;
const mem = std.mem;
const heap = std.heap;
const process = std.process;

const SOCK_PATH = "/run/coco/visor.sock";
const DEFAULT_MEMORY_MB: u32 = 512;
const DEFAULT_VCPU: u32 = 2;

const REQ_BOOT: u32 = 1;
const REQ_DESTROY: u32 = 3;
const REQ_GET_STATE: u32 = 6;

const VisorHeader = extern struct {
    kind: u32,
    size: u32,
};

const VisorBootRequest = struct {
    rootfs_path_len: u32,
    memory_mb: u32,
    vcpu_count: u32,
    kernel_path_len: u32,
    initrd_path_len: u32,
    sandbox_id_len: u32,
    vsock_port: u32,
    padding: u32,
};

const VisorBootResponse = struct {
    success: bool,
    vsock_cid: u32,
    vsock_port: u32,
    pid: u64,
    error_msg: [256]u8,
};

const VisorDestroyRequest = struct {
    force: bool,
    sandbox_id_len: u32,
};

const VisorGetStateResponse = struct {
    state: u32,
    pid: u64,
    vsock_cid: u32,
};

fn connectVisor(sock_path: []const u8) !net.Stream {
    return net.connectUnixSocket(sock_path);
}

fn bootVm(sandbox_id: []const u8, rootfs_path: []const u8) !VisorBootResponse {
    const sock = try connectVisor(SOCK_PATH);
    defer sock.close();

    var req = VisorBootRequest{
        .rootfs_path_len = @intCast(rootfs_path.len),
        .memory_mb = DEFAULT_MEMORY_MB,
        .vcpu_count = DEFAULT_VCPU,
        .kernel_path_len = 0,
        .initrd_path_len = 0,
        .sandbox_id_len = @intCast(sandbox_id.len),
        .vsock_port = 0,
        .padding = 0,
    };

    var header = VisorHeader{ .kind = REQ_BOOT, .size = @sizeOf(VisorBootRequest) + rootfs_path.len + sandbox_id.len };
    try sock.writeAll(mem.asBytes(&header));
    try sock.writeAll(mem.asBytes(&req));
    try sock.writeAll(rootfs_path);
    try sock.writeAll(sandbox_id);

    var resp: VisorBootResponse = undefined;
    const n = try sock.read(mem.asBytes(&resp));
    if (n != @sizeOf(VisorBootResponse)) return error.UnexpectedResponse;
    return resp;
}

fn destroyVm(sandbox_id: []const u8) !void {
    const sock = try connectVisor(SOCK_PATH);
    defer sock.close();

    var header = VisorHeader{ .kind = REQ_DESTROY, .size = @sizeOf(VisorDestroyRequest) + sandbox_id.len };
    var req = VisorDestroyRequest{ .force = true, .sandbox_id_len = @intCast(sandbox_id.len) };

    try sock.writeAll(mem.asBytes(&header));
    try sock.writeAll(mem.asBytes(&req));
    try sock.writeAll(sandbox_id);
}

fn getVmState() !VisorGetStateResponse {
    const sock = try connectVisor(SOCK_PATH);
    defer sock.close();

    var header = VisorHeader{ .kind = REQ_GET_STATE, .size = 0 };
    try sock.writeAll(mem.asBytes(&header));

    var resp: VisorGetStateResponse = undefined;
    const n = try sock.read(mem.asBytes(&resp));
    if (n != @sizeOf(VisorGetStateResponse)) return error.UnexpectedResponse;
    return resp;
}

pub fn main() !void {
    const allocator = heap.c_allocator;
    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    if (args.len < 2) {
        try std.io.getStdOut().writeAll("Usage: cocovisor <subcommand>\n");
        try std.io.getStdOut().writeAll("Subcommands: boot, exec, destroy, state\n");
        return;
    }

    const subcmd = args[1];

    if (mem.eql(u8, subcmd, "boot")) {
        if (args.len < 4) {
            try std.io.getStdOut().writeAll("Usage: cocovisor boot <sandbox_id> <rootfs_path>\n");
            return;
        }
        const sandbox_id = args[2];
        const rootfs_path = args[3];
        const resp = try bootVm(sandbox_id, rootfs_path);

        if (!resp.success) {
            try std.io.getStdOut().writeAll("Boot failed\n");
            return error.BootFailed;
        }

        try std.fmt.print("Booted sandbox {s}: pid={}, vsock_cid={}\n", .{
            sandbox_id, resp.pid, resp.vsock_cid,
        });
    } else if (mem.eql(u8, subcmd, "destroy")) {
        if (args.len < 3) {
            try std.io.getStdOut().writeAll("Usage: cocovisor destroy <sandbox_id>\n");
            return;
        }
        const sandbox_id = args[2];
        try destroyVm(sandbox_id);
        try std.fmt.print("Destroyed sandbox {s}\n", .{sandbox_id});
    } else if (mem.eql(u8, subcmd, "state")) {
        const resp = try getVmState();
        try std.fmt.print("State: {}, pid={}, vsock_cid={}\n", .{ resp.state, resp.pid, resp.vsock_cid });
    } else {
        try std.fmt.print("Unknown subcommand: {s}\n", .{subcmd});
        return error.InvalidCommand;
    }
}
