// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

const VSOCK_CID_ANY: u32 = 0xFFFFFFFF;
const VSOCK_CID_HOST: u32 = 2;

pub const VsockError = error{
    SocketFailed,
    ConnectFailed,
    BindFailed,
    ListenFailed,
    AcceptFailed,
};

pub const Vsock = struct {
    fd: posix.fd_t,
    cid: u32,
    port: u32,

    pub fn socket() VsockError!Vsock {
        const fd = posix.socket(posix.AF.VSOCK, posix.SOCK.STREAM, 0);
        if (fd < 0) {
            return VsockError.SocketFailed;
        }

        return Vsock{
            .fd = fd,
            .cid = 0,
            .port = 0,
        };
    }

    pub fn close(self: *Vsock) void {
        posix.close(self.fd);
    }

    pub fn connect(self: *Vsock, cid: u32, port: u32) VsockError!void {
        var addr: posix.sockaddr_vsock = .{
            .family = posix.AF.VSOCK,
            .reserved = 0,
            .src_cid = 0,
            .src_port = 0,
            .dst_cid = cid,
            .dst_port = port,
        };

        const result = posix.connect(self.fd, @ptrCast(&addr), @sizeOf(posix.sockaddr_vsock));
        if (result != 0) {
            return VsockError.ConnectFailed;
        }

        self.cid = cid;
        self.port = port;
    }

    pub fn bind(self: *Vsock, port: u32) VsockError!void {
        var addr: posix.sockaddr_vsock = .{
            .family = posix.AF.VSOCK,
            .reserved = 0,
            .src_cid = VSOCK_CID_ANY,
            .src_port = port,
            .dst_cid = 0,
            .dst_port = 0,
        };

        const result = posix.bind(self.fd, @ptrCast(&addr), @sizeOf(posix.sockaddr_vsock));
        if (result != 0) {
            return VsockError.BindFailed;
        }

        self.port = port;
    }

    pub fn listen(self: *Vsock, backlog: u32) VsockError!void {
        const result = posix.listen(self.fd, backlog);
        if (result != 0) {
            return VsockError.ListenFailed;
        }
    }

    pub fn accept(self: *Vsock) VsockError!Vsock {
        var client_addr: posix.sockaddr_vsock = undefined;
        var addr_len: posix.socklen_t = @sizeOf(posix.sockaddr_vsock);

        const client_fd = posix.accept(self.fd, @ptrCast(&client_addr), &addr_len);
        if (client_fd < 0) {
            return VsockError.AcceptFailed;
        }

        return Vsock{
            .fd = client_fd,
            .cid = client_addr.dst_cid,
            .port = client_addr.dst_port,
        };
    }

    pub fn send(self: *Vsock, data: []const u8) !usize {
        return posix.send(self.fd, data, 0);
    }

    pub fn recv(self: *Vsock, buffer: []u8) !usize {
        return posix.recv(self.fd, buffer, 0);
    }
};

pub fn createVsockPair() VsockError!struct { client: Vsock, server: Vsock } {
    var fds: [2]posix.fd_t = undefined;
    const result = posix.socketpair(posix.AF.VSOCK, posix.SOCK.STREAM, 0, &fds);
    if (result != 0) {
        return VsockError.SocketFailed;
    }

    return .{
        .client = Vsock{ .fd = fds[0], .cid = 0, .port = 0 },
        .server = Vsock{ .fd = fds[1], .cid = 0, .port = 0 },
    };
}
