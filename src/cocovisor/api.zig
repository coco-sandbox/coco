// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const net = std.net;
const os = std.os;
const mem = std.mem;
const heap = std.heap;
const fs = std.fs;

const REQ_BOOT: u32 = 1;
const REQ_GET_STATE: u32 = 6;
const SOCK_PATH = "/run/coco/visor.sock";

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

const VisorGetStateResponse = struct {
    state: u32,
    pid: u64,
    vsock_cid: u32,
};

pub fn main() !void {
    os.mkdir("/run/coco", 0o755) catch {};
    fs.deleteFileAbsolute(SOCK_PATH) catch {};

    const listener = try net.listenUnix(SOCK_PATH);
    defer listener.close();
    try std.fmt.print("cocovisor listening on {s}\n", .{SOCK_PATH});

    while (true) {
        const client = listener.accept() catch continue;
        defer client.close();

        var header_buf: [8]u8 = undefined;
        const n = client.read(header_buf[0..]) catch continue;
        if (n < 8) continue;

        var le = std.mem.readIntLittle(u32, header_buf[0..4]);
        var size = std.mem.readIntLittle(u32, header_buf[4..8]);

        _ = size;

        switch (le) {
            REQ_BOOT => {
                var resp = VisorBootResponse{
                    .success = true,
                    .vsock_cid = 3,
                    .vsock_port = 0,
                    .pid = 12345,
                    .error_msg = [_]u8{0} ** 256,
                };
                _ = client.write(mem.asBytes(&resp)) catch continue;
            },
            REQ_GET_STATE => {
                var state_resp = VisorGetStateResponse{
                    .state = 1,
                    .pid = 0,
                    .vsock_cid = 0,
                };
                _ = client.write(mem.asBytes(&state_resp)) catch continue;
            },
            else => {},
        }
    }
}
