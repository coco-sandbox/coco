const std = @import("std");
const linux = std.os.linux;

pub const O = linux.O;
pub const PROT = linux.PROT;
pub const MAP = linux.MAP;
pub const SIG = linux.SIG;
pub const E = linux.E;

pub fn open(path: []const u8, flags: linux.O, mode: u32) !i32 {
    var path_buf: [4096]u8 = undefined;
    if (path.len >= path_buf.len) return error.PathTooLong;
    @memcpy(path_buf[0..path.len], path);
    path_buf[path.len] = 0;
    const z: [*:0]const u8 = @ptrCast(&path_buf);
    const rc = linux.openat(linux.AT.FDCWD, z, flags, mode);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.OpenFailed;
    return @intCast(rc);
}

pub fn openZ(path: [*:0]const u8, flags: linux.O, mode: u32) !i32 {
    const rc = linux.openat(linux.AT.FDCWD, path, flags, mode);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.OpenFailed;
    return @intCast(rc);
}

pub fn close(fd: i32) void {
    _ = linux.close(fd);
}

pub fn read(fd: i32, buf: []u8) !usize {
    const rc = linux.read(fd, buf.ptr, buf.len);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.ReadFailed;
    return @intCast(rc);
}

pub fn write(fd: i32, buf: []const u8) !usize {
    const rc = linux.write(fd, buf.ptr, buf.len);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.WriteFailed;
    return @intCast(rc);
}

pub fn pwrite(fd: i32, buf: []const u8, offset: i64) !usize {
    const rc = linux.pwrite(fd, buf.ptr, buf.len, offset);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.PwriteFailed;
    return @intCast(rc);
}

pub fn lseek(fd: i32, offset: i64, whence: u32) !i64 {
    const rc = linux.lseek(fd, offset, whence);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.LseekFailed;
    return @intCast(rc);
}

pub fn ftruncate(fd: i32, length: i64) !void {
    const rc = linux.ftruncate(fd, length);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.FtruncateFailed;
}

pub fn fsync(fd: i32) !void {
    const rc = linux.fsync(fd);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.FsyncFailed;
}

pub fn unlink(path: []const u8) !void {
    var path_buf: [4096]u8 = undefined;
    if (path.len >= path_buf.len) return error.PathTooLong;
    @memcpy(path_buf[0..path.len], path);
    path_buf[path.len] = 0;
    const z: [*:0]const u8 = @ptrCast(&path_buf);
    const rc = linux.unlinkat(linux.AT.FDCWD, z, 0);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.UnlinkFailed;
}

pub fn mkdir(path: []const u8, mode: u32) !void {
    var path_buf: [4096]u8 = undefined;
    if (path.len >= path_buf.len) return error.PathTooLong;
    @memcpy(path_buf[0..path.len], path);
    path_buf[path.len] = 0;
    const z: [*:0]const u8 = @ptrCast(&path_buf);
    const rc = linux.mkdirat(linux.AT.FDCWD, z, mode);
    const sr: isize = @bitCast(rc);
    if (sr < 0 and -sr != @intFromEnum(linux.E.EXIST)) return error.MkdirFailed;
}

pub fn nanoTimestamp() i128 {
    var ts: linux.timespec = undefined;
    _ = linux.clock_gettime(linux.CLOCK.MONOTONIC, &ts);
    return @as(i128, ts.sec) * 1_000_000_000 + ts.nsec;
}

pub fn timestamp() i64 {
    var ts: linux.timespec = undefined;
    _ = linux.clock_gettime(linux.CLOCK.REALTIME, &ts);
    return ts.sec;
}

pub fn sleep(nanoseconds: u64) void {
    var ts: linux.timespec = .{
        .sec = @intCast(nanoseconds / 1_000_000_000),
        .nsec = @intCast(nanoseconds % 1_000_000_000),
    };
    _ = linux.nanosleep(&ts, &ts);
}

pub fn fileExists(path: []const u8) bool {
    var path_buf: [4096]u8 = undefined;
    if (path.len >= path_buf.len) return false;
    @memcpy(path_buf[0..path.len], path);
    path_buf[path.len] = 0;
    const z: [*:0]const u8 = @ptrCast(&path_buf);
    var stx: linux.Statx = undefined;
    const rc = linux.statx(linux.AT.FDCWD, z, 0, .{ .SIZE = true }, &stx);
    const sr: isize = @bitCast(rc);
    return sr == 0;
}

pub fn fileSize(fd: i32) !u64 {
    var stx: linux.Statx = undefined;
    const rc = linux.statx(fd, "", linux.AT.EMPTY_PATH, .{ .SIZE = true }, &stx);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.StatFailed;
    return stx.size;
}

const Statfs64 = extern struct {
    f_type: i64,
    f_bsize: i64,
    f_blocks: u64,
    f_bfree: u64,
    f_bavail: u64,
    f_files: u64,
    f_ffree: u64,
    f_fsid: [2]i32,
    f_namelen: i64,
    f_frsize: i64,
    f_flags: i64,
    f_spare: [4]i64,
};

pub fn statfsType(path: []const u8) u64 {
    var path_buf: [4096]u8 = undefined;
    if (path.len >= path_buf.len) return 0;
    @memcpy(path_buf[0..path.len], path);
    path_buf[path.len] = 0;
    var st: Statfs64 = undefined;
    const SYS_statfs: usize = 137;
    const rc = linux.syscall2(@enumFromInt(SYS_statfs), @intFromPtr(&path_buf), @intFromPtr(&st));
    const sr: isize = @bitCast(rc);
    if (sr < 0) return 0;
    return @bitCast(st.f_type);
}

pub fn copyFile(src: []const u8, dst: []const u8) !void {
    const src_fd = try open(src, .{ .ACCMODE = .RDONLY }, 0);
    defer close(src_fd);
    const dst_fd = try open(dst, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644);
    defer close(dst_fd);
    var buf: [65536]u8 = undefined;
    while (true) {
        const n = try read(src_fd, &buf);
        if (n == 0) break;
        var written: usize = 0;
        while (written < n) {
            written += try write(dst_fd, buf[written..n]);
        }
    }
}

pub fn copyFileRange(src_fd: i32, dst_fd: i32, len: u64) !u64 {
    const SYS_copy_file_range: usize = 326;
    var copied: u64 = 0;
    while (copied < len) {
        const remaining = len - copied;
        const rc = linux.syscall6(@enumFromInt(SYS_copy_file_range), @as(usize, @intCast(src_fd)), 0, @as(usize, @intCast(dst_fd)), 0, @intCast(remaining), 0);
        const sr: isize = @bitCast(rc);
        if (sr < 0) return error.CopyFileRangeFailed;
        if (sr == 0) break;
        copied += @intCast(sr);
    }
    return copied;
}

pub const DirEntry = struct {
    name: [256]u8,
    name_len: usize,
    kind: u8,

    pub fn nameSlice(self: *const DirEntry) []const u8 {
        return self.name[0..self.name_len];
    }
};

pub const DirIterator = struct {
    fd: i32,
    buf: [4096]u8 = undefined,
    buf_len: usize = 0,
    buf_pos: usize = 0,

    const linux_dirent64 = extern struct {
        d_ino: u64,
        d_off: i64,
        d_reclen: u16,
        d_type: u8,
        d_name: u8,
    };

    pub fn deinit(self: *DirIterator) void {
        close(self.fd);
    }

    pub fn next(self: *DirIterator) !?DirEntry {
        while (true) {
            if (self.buf_pos >= self.buf_len) {
                const rc = linux.syscall3(.getdents64, @as(usize, @intCast(self.fd)), @intFromPtr(&self.buf), self.buf.len);
                const sr: isize = @bitCast(rc);
                if (sr < 0) return error.GetdentsFailed;
                if (sr == 0) return null;
                self.buf_len = @intCast(sr);
                self.buf_pos = 0;
            }
            const ent_ptr: *align(1) linux_dirent64 = @ptrCast(&self.buf[self.buf_pos]);
            const reclen = ent_ptr.d_reclen;
            const name_ptr: [*]const u8 = @ptrCast(&ent_ptr.d_name);
            var name_len: usize = 0;
            while (name_len < reclen and name_ptr[name_len] != 0) : (name_len += 1) {}
            self.buf_pos += reclen;
            if (name_len == 1 and name_ptr[0] == '.') continue;
            if (name_len == 2 and name_ptr[0] == '.' and name_ptr[1] == '.') continue;
            var ent: DirEntry = .{ .name = undefined, .name_len = name_len, .kind = ent_ptr.d_type };
            @memcpy(ent.name[0..name_len], name_ptr[0..name_len]);
            return ent;
        }
    }
};

pub fn openDir(path: []const u8) !DirIterator {
    var path_buf: [4096]u8 = undefined;
    if (path.len >= path_buf.len) return error.PathTooLong;
    @memcpy(path_buf[0..path.len], path);
    path_buf[path.len] = 0;
    const z: [*:0]const u8 = @ptrCast(&path_buf);
    const rc = linux.openat(linux.AT.FDCWD, z, .{ .ACCMODE = .RDONLY, .DIRECTORY = true }, 0);
    const sr: isize = @bitCast(rc);
    if (sr < 0) return error.OpenDirFailed;
    return DirIterator{ .fd = @intCast(rc) };
}

pub const DT_REG: u8 = 8;
pub const DT_DIR: u8 = 4;
pub const BTRFS_SUPER_MAGIC: u64 = 0x9123683E;
