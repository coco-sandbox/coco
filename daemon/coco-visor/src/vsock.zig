const std = @import("std");
const sc = @import("syscall.zig");
const linux = std.os.linux;
const posix = std.posix;

const AF_VSOCK = 40;
const VMADDR_CID_ANY: u32 = 0xFFFFFFFF;

const VHOST_VSOCK_SET_GUEST_CID: u32 = 0x4008AF60;

pub const SockaddrVm = extern struct {
    svm_family: u16,
    svm_reserved1: u16,
    svm_port: u32,
    svm_cid: u32,
    svm_flags: u8,
    svm_zero: [3]u8,
};

pub fn openAndAssignCid(cid: u32) !i32 {
    const fd = try sc.open("/dev/vhost-vsock", .{ .ACCMODE = .RDWR }, 0);
    errdefer sc.close(fd);

    var cid64: u64 = cid;
    const rc = linux.ioctl(fd, VHOST_VSOCK_SET_GUEST_CID, @intFromPtr(&cid64));
    if (@as(isize, @bitCast(rc)) < 0)
        return error.VhostVsockSetCidFailed;

    return fd;
}

pub fn createServer(port: u32) !i32 {
    const sock_rc = linux.socket(AF_VSOCK, linux.SOCK.STREAM, 0);
    if (@as(isize, @bitCast(sock_rc)) < 0) return error.SocketCreateFailed;
    const fd: i32 = @intCast(sock_rc);
    errdefer _ = linux.close(fd);

    const addr = SockaddrVm{
        .svm_family = AF_VSOCK,
        .svm_reserved1 = 0,
        .svm_port = port,
        .svm_cid = VMADDR_CID_ANY,
        .svm_flags = 0,
        .svm_zero = .{ 0, 0, 0 },
    };

    const bind_rc = linux.bind(fd, @ptrCast(&addr), @sizeOf(SockaddrVm));
    if (@as(isize, @bitCast(bind_rc)) < 0) return error.BindFailed;

    const listen_rc = linux.listen(fd, 16);
    if (@as(isize, @bitCast(listen_rc)) < 0) return error.ListenFailed;

    return fd;
}

pub fn acceptAgent(server_fd: i32) !struct { fd: i32, guest_cid: u32 } {
    var addr: SockaddrVm = undefined;
    var addr_len: u32 = @sizeOf(SockaddrVm);

    const rc = linux.accept(server_fd, @ptrCast(&addr), &addr_len);
    if (@as(isize, @bitCast(rc)) <= 0) return error.AcceptFailed;
    const accepted_fd: i32 = @intCast(rc);
    return .{
        .fd = accepted_fd,
        .guest_cid = addr.svm_cid,
    };
}

pub fn connectToAgent(guest_cid: u32, port: u32) !i32 {
    const sock_rc = linux.socket(AF_VSOCK, linux.SOCK.STREAM, 0);
    if (@as(isize, @bitCast(sock_rc)) < 0) return error.SocketCreateFailed;
    const fd: i32 = @intCast(sock_rc);
    errdefer _ = linux.close(fd);

    const addr = SockaddrVm{
        .svm_family = AF_VSOCK,
        .svm_reserved1 = 0,
        .svm_port = port,
        .svm_cid = guest_cid,
        .svm_flags = 0,
        .svm_zero = .{ 0, 0, 0 },
    };

    const connect_rc = linux.connect(fd, @ptrCast(&addr), @sizeOf(SockaddrVm));
    if (@as(isize, @bitCast(connect_rc)) < 0) return error.ConnectFailed;
    return fd;
}
