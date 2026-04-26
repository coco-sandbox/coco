// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Cloud Hypervisor client via UNIX domain socket.
//! Handles process spawning, HTTP/JSON API communication, and lifecycle.

const std = @import("std");

pub const CLHError = error{
    ConnectionFailed,
    CommandFailed,
    Timeout,
    InvalidResponse,
    SpawnFailed,
    ProcessExited,
    IoError,
    OutOfMemory,
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

fn dupePath(allocator: std.mem.Allocator, path: []const u8) []const u8 {
    const dup = allocator.alloc(u8, path.len) catch return path;
    @memcpy(dup, path);
    return dup;
}

pub const CLHClient = struct {
    sock_path: []const u8,
    pid: u32,
    sock: ?std.net.Stream = null,
    allocator: std.mem.Allocator,

    /// Spawn a new Cloud Hypervisor process and connect to its API socket.
    /// This forks the current process and execs cloud-hypervisor.
    pub fn spawn(vm_id: []const u8, config: *const VMConfig, allocator: std.mem.Allocator) !CLHClient {
        // Create socket directory
        const sock_dir = std.fmt.allocPrint(allocator, "/run/coco/vm/{s}", .{vm_id}) catch return CLHError.OutOfMemory;
        std.fs.makeDirAbsolute(sock_dir) catch {};

        const api_sock_path = std.fmt.allocPrint(allocator, "/run/coco/vm/{s}/sock", .{vm_id}) catch return CLHError.OutOfMemory;
        const vsock_path = std.fmt.allocPrint(allocator, "/run/coco/vm/{s}/vsock.sock", .{vm_id}) catch return CLHError.OutOfMemory;

        // Build CLH argv
        var argv = std.ArrayList([]const u8).init(allocator);
        defer argv.deinit();

        try argv.append("/usr/bin/cloud-hypervisor");
        try argv.append("--api-socket");
        try argv.append(api_sock_path);
        try argv.append("--kernel");
        try argv.append(config.kernel);
        try argv.append("--disk");
        try argv.append(try std.fmt.allocPrint(allocator, "path={s}", .{config.rootfs}));
        try argv.append("--memory");
        try argv.append(try std.fmt.allocPrint(allocator, "{d}M", .{config.memory_mb}));
        try argv.append("--vcpus");
        try argv.append(try std.fmt.allocPrint(allocator, "{d}", .{config.vcpus}));
        try argv.append("--vsock");
        try argv.append(try std.fmt.allocPrint(allocator, "cid={d},socket={s}", .{config.vsock_cid, vsock_path}));

        if (config.initrd.len > 0) {
            try argv.append("--initramfs");
            try argv.append(config.initrd);
        }

        // Fork child process
        const pid = std.posix.fork() catch return CLHError.SpawnFailed;
        if (pid == 0) {
            // Child: exec cloud-hypervisor
            const argv_ptrs = allocator.alloc(?[*:0]const u8, argv.items.len + 1) catch std.posix.exit(1);
            defer allocator.free(argv_ptrs);
            for (argv.items, 0..) |arg, i| {
                argv_ptrs[i] = @ptrCast(arg);
            }
            argv_ptrs[argv.items.len] = null;

            // execveZ expects [*:null]const ?[*]const u8 - null-terminated array of optional C strings
            const envp = [1]?[*:0]const u8{null};
            const path: [*:0]const u8 = @ptrCast(argv_ptrs[0].?);
            const child_argv: [*:null]const ?[*:0]const u8 = @ptrCast(argv_ptrs);
            const env: [*:null]const ?[*:0]const u8 = @ptrFromInt(@intFromPtr(&envp));
            std.posix.execveZ(path, child_argv, env) catch {
                std.debug.print("[clh] Failed to exec cloud-hypervisor\n", .{});
                std.posix.exit(1);
            };
            unreachable;
        }

        // Parent: wait for API socket to be ready (30s timeout)
        try waitForSocket(api_sock_path, 30000);

        // Connect to the API socket
        var client = CLHClient{
            .sock_path = dupePath(allocator, api_sock_path),
            .pid = @as(u32, @intCast(pid)),
            .sock = try std.net.connectUnixSocket(api_sock_path),
            .allocator = allocator,
        };

        // Send boot command via HTTP/JSON
        const boot_result = client.boot(config);
        if (boot_result) |_| {} else |_| {}

        return client;
    }

    /// Wait for the API socket to become available (CLH is ready).
    fn waitForSocket(sock_path: []const u8, timeout_ms: u32) !void {
        const deadline = @as(u64, @intCast(timeout_ms)) * 1_000_000;
        const start = std.time.nanoTimestamp();

        while (true) {
            _ = std.posix.open(sock_path, .{ .ACCMODE = .RDONLY }, 0o000) catch {
                // Socket not ready yet, check timeout
                if (std.time.nanoTimestamp() - start > deadline) {
                    return CLHError.Timeout;
                }
                std.time.sleep(10_000_000); // 10ms
                continue;
            };
            // Give CLH a moment to finish initialization
            std.time.sleep(50_000_000); // 50ms
            return;
        }
    }

    pub fn disconnect(self: *CLHClient) void {
        if (self.sock) |s| s.close();
        self.sock = null;
    }

    /// Kill the CLH process and reap its zombie.
    pub fn kill(self: *CLHClient) void {
        if (self.pid == 0) return;

        const pid: i32 = @intCast(self.pid);

        // Try graceful shutdown first via SIGTERM
        std.posix.kill(pid, std.posix.SIG.TERM) catch {};

        // Wait with timeout (5s)
        const deadline = std.time.nanoTimestamp() + 5_000_000_000;
        while (std.time.nanoTimestamp() < deadline) {
            const ret = std.posix.waitpid(pid, 1); // 1 = WNOHANG
            if (ret.pid != 0) {
                // Process reaped or error
                self.pid = 0;
                return;
            }
            std.time.sleep(50_000_000);
        }

        // Force kill if still alive
        std.posix.kill(pid, std.posix.SIG.KILL) catch {};
        _ = std.posix.waitpid(pid, 0);
        self.pid = 0;
    }

    pub fn boot(self: *CLHClient, config: *const VMConfig) !BootResult {
        const id = config.id;
        const kernel = config.kernel;
        const initrd = config.initrd;
        const rootfs = config.rootfs;
        const vcpus = config.vcpus;
        const memory_mb = config.memory_mb;

        // Build boot request JSON matching CLH HTTP API
        const json = try std.fmt.allocPrint(self.allocator,
            \\{{"id":"{s}","boot_source":{{"kernel":"{s}","initramfs":"{s}"}},"root_volume":{{"path":"{s}","readonly":true}},"cpus":{{"count":{d}}},"memory":{{"size":"{d}M"}}}}
        , .{id, kernel, initrd, rootfs, vcpus, memory_mb});

        _ = try self.sendCommand("vm.create", json);
        self.allocator.free(json);

        // Get VM info to retrieve real PID
        const info = try self.sendCommand("vm.info", "");
        defer self.allocator.free(info);

        const pid = parseIntFromJson(info, "pid", self.allocator);

        return BootResult{
            .pid = pid,
            .vsock_cid = config.vsock_cid,
        };
    }

    /// Send HTTP/JSON command to CLH API socket.
    /// Format: POST /api/v1/{command} HTTP/1.1\r\nHost: localhost\r\nAccept: */*\r\nContent-Length: {len}\r\n\r\n{body}
    fn sendCommand(self: *CLHClient, command: []const u8, body: []const u8) ![]u8 {
        const path = try std.fmt.allocPrint(self.allocator, "/api/v1/{s}", .{command});
        defer self.allocator.free(path);

        var http = std.ArrayList(u8).init(self.allocator);
        defer http.deinit();

        try http.writer().print(
            "POST {s} HTTP/1.1\r\nHost: localhost\r\nAccept: */*\r\nContent-Length: {d}\r\n\r\n",
            .{ path, body.len }
        );
        if (body.len > 0) {
            try http.appendSlice(body);
        }

        try self.sock.?.writeAll(http.items);

        return self.readHttpResponse();
    }

    fn readHttpResponse(self: *CLHClient) ![]u8 {
        var header_buf: [1024]u8 = undefined;
        const header_len = try self.sock.?.readAll(&header_buf);
        if (header_len == 0) return CLHError.ConnectionFailed;

        const header = header_buf[0..header_len];

        // Find end of headers (\r\n\r\n)
        const header_end = std.mem.indexOf(u8, header, "\r\n\r\n") orelse return CLHError.InvalidResponse;
        const headers = header[0..header_end];

        // Parse Content-Length
        const content_len = parseContentLength(headers);
        if (content_len == 0) {
            return self.allocator.dupe(u8, "");
        }

        // Read body
        var body = try self.allocator.alloc(u8, content_len);
        errdefer self.allocator.free(body);
        var offset: usize = 0;
        while (offset < content_len) {
            const n = try self.sock.?.read(body[offset..content_len]);
            if (n == 0) break;
            offset += n;
        }

        return body;
    }

    pub fn shutdown(self: *CLHClient, vm_id: []const u8) !void {
        const cmd = try std.fmt.allocPrint(self.allocator,
            \\{{"id":"{s}","action":"Shutdown"}}
        , .{vm_id});
        defer self.allocator.free(cmd);

        _ = try self.sendCommand("vm.shutdown", cmd);
    }

    pub fn pause(self: *CLHClient, vm_id: []const u8) !void {
        const cmd = try std.fmt.allocPrint(self.allocator,
            \\{{"id":"{s}","action":"Pause"}}
        , .{vm_id});
        defer self.allocator.free(cmd);

        _ = try self.sendCommand("vm.pause", cmd);
    }

    pub fn resumeVm(self: *CLHClient, vm_id: []const u8) !void {
        const cmd = try std.fmt.allocPrint(self.allocator,
            \\{{"id":"{s}","action":"Resume"}}
        , .{vm_id});
        defer self.allocator.free(cmd);

        _ = try self.sendCommand("vm.resume", cmd);
    }
};

fn parseContentLength(headers: []const u8) usize {
    const prefix = "Content-Length: ";
    const start = std.mem.indexOf(u8, headers, prefix) orelse return 0;
    const val_start = start + prefix.len;
    var val_end = val_start;
    while (val_end < headers.len and headers[val_end] >= '0' and headers[val_end] <= '9') {
        val_end += 1;
    }
    if (val_end <= val_start) return 0;

    var result: usize = 0;
    for (headers[val_start..val_end]) |c| {
        result = result * 10 + (c - '0');
    }
    return result;
}

fn parseIntFromJson(json: []const u8, field: []const u8, allocator: std.mem.Allocator) u32 {
    const prefix = std.fmt.allocPrint(allocator, "\"{s}\":", .{field}) catch return 0;
    defer allocator.free(prefix);

    const start = std.mem.indexOf(u8, json, prefix) orelse return 0;
    const val_start = start + prefix.len;

    var val_end = val_start;
    while (val_end < json.len and json[val_end] >= '0' and json[val_end] <= '9') {
        val_end += 1;
    }

    if (val_end <= val_start) return 0;

    var result: u32 = 0;
    for (json[val_start..val_end]) |c| {
        result = result * 10 + (c - '0');
    }
    return result;
}

pub const BootResult = struct {
    pid: u32,
    vsock_cid: u32,
};