// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cloud Hypervisor client via UNIX domain socket

const std = @import("std");

pub const CLHError = error{
    ConnectionFailed,
    CommandFailed,
    Timeout,
    InvalidResponse,
};

pub const CLHClient = struct {
    sock_path: []const u8,
    sock: ?std.net.Stream = null,

    pub fn connect(path: []const u8) !CLHClient {
        var client = CLHClient{ .sock_path = path };
        client.sock = try std.net.Dial.connect(std.net.Address.initUnix(path));
        return client;
    }

    pub fn disconnect(self: *CLHClient) void {
        if (self.sock) |s| s.close();
        self.sock = null;
    }

    pub fn boot(self: *CLHClient, config: *const VMConfig) !BootResult {
        const id = config.id;
        const kernel = config.kernel;
        const initrd = config.initrd;
        const rootfs = config.rootfs;
        const vcpus = config.vcpus;
        const memory_mb = config.memory_mb;

        // Build boot request JSON
        var json_buf: [512]u8 = undefined;
        const json = std.fmt.bufPrint(&json_buf,
            \\{{"id":"{s}","boot_source":{{"kernel":"{s}","initramfs":"{s}"}},"root_volume":{{"path":"{s}","readonly":true}},"cpus":{{"count":{d}}},"memory":{{"size":"{d}M"}}}}
        , .{id, kernel, initrd, rootfs, vcpus, memory_mb}) catch return CLHError.CommandFailed;

        try self.sendRaw(json);

        const resp = try self.recvRaw();
        defer std.heap.page_allocator.free(resp);

        // Parse response JSON: {"id": "...", "pid": 1234, "vsock_cid": 3}
        // Simple parser: find "pid": and "vsock_cid": values
        var pid: u32 = 0;
        var vsock_cid: u32 = config.vsock_cid;

        // Find pid value
        const pid_start = std.mem.indexOf(u8, resp, "\"pid\":") orelse return BootResult{
            .pid = 0,
            .vsock_cid = vsock_cid,
        };
        const pid_val_start = pid_start + 6;
        var pid_val_end = pid_val_start;
        while (pid_val_end < resp.len and resp[pid_val_end] >= '0' and resp[pid_val_end] <= '9') {
            pid_val_end += 1;
        }
        if (pid_val_end > pid_val_start) {
            var pid_val: u32 = 0;
            for (resp[pid_val_start..pid_val_end]) |c| {
                pid_val = pid_val * 10 + (c - '0');
            }
            pid = pid_val;
        }

        // Find vsock_cid value
        const vsock_start = std.mem.indexOf(u8, resp, "\"vsock_cid\":") orelse return BootResult{
            .pid = pid,
            .vsock_cid = vsock_cid,
        };
        const vsock_val_start = vsock_start + 12;
        var vsock_val_end = vsock_val_start;
        while (vsock_val_end < resp.len and resp[vsock_val_end] >= '0' and resp[vsock_val_end] <= '9') {
            vsock_val_end += 1;
        }
        if (vsock_val_end > vsock_val_start) {
            var vsock_val: u32 = 0;
            for (resp[vsock_val_start..vsock_val_end]) |c| {
                vsock_val = vsock_val * 10 + (c - '0');
            }
            vsock_cid = vsock_val;
        }

        return BootResult{
            .pid = pid,
            .vsock_cid = vsock_cid,
        };
    }

    pub fn shutdown(self: *CLHClient, vm_id: []const u8) !void {
        var cmd_buf: [256]u8 = undefined;
        const cmd = std.fmt.bufPrint(&cmd_buf,
            \\{{"id":"{s}","action":"Shutdown"}}
        , .{vm_id}) catch return CLHError.CommandFailed;

        try self.sendRaw(cmd);
        _ = try self.recvRaw();
    }

    pub fn pause(self: *CLHClient, vm_id: []const u8) !void {
        var cmd_buf: [256]u8 = undefined;
        const cmd = std.fmt.bufPrint(&cmd_buf,
            \\{{"id":"{s}","action":"Pause"}}
        , .{vm_id}) catch return CLHError.CommandFailed;

        try self.sendRaw(cmd);
    }

    pub fn resume(self: *CLHClient, vm_id: []const u8) !void {
        var cmd_buf: [256]u8 = undefined;
        const cmd = std.fmt.bufPrint(&cmd_buf,
            \\{{"id":"{s}","action":"Resume"}}
        , .{vm_id}) catch return CLHError.CommandFailed;

        try self.sendRaw(cmd);
    }

    fn sendRaw(self: *CLHClient, msg: []const u8) !void {
        const frame = try std.heap.page_allocator.alloc(u8, 8 + msg.len);
        defer std.heap.page_allocator.free(frame);
        std.mem.writeInt(u32, frame[0..4].*, @intCast(msg.len), .little);
        @memcpy(frame[8..], msg);
        try self.sock.?.writeAll(frame);
    }

    fn recvRaw(self: *CLHClient) ![]u8 {
        var header: [8]u8 = undefined;
        _ = try self.sock.?.read(header[0..8]);
        const size = std.mem.readInt(u32, header[0..4], .little);
        var data = try std.heap.page_allocator.alloc(u8, size);
        _ = try self.sock.?.readAll(data);
        return data;
    }
};

pub const BootResult = struct {
    pid: u32,
    vsock_cid: u32,
};

pub const VMConfig = struct {
    id: []const u8,
    rootfs: []const u8,
    kernel: []const u8 = "/var/lib/coco/vmlinux",
    initrd: []const u8 = "",
    memory_mb: u32 = 512,
    vcpus: u32 = 2,
    vsock_cid: u32,
    tap_name: []const u8 = "",
};