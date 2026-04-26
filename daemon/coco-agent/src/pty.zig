// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

const PTYError = error{
    OpenFailed,
    GrantFailed,
    UnlockFailed,
    SetTermiosFailed,
};

pub const PTY = struct {
    master: posix.fd_t,
    slave: posix.fd_t,
    slave_name: []u8,

    pub fn open() PTYError!PTY {
        var master: posix.fd_t = undefined;
        var slave: posix.fd_t = undefined;
        var slave_name: [128]u8 = undefined;

        const result = posix.openpty(&master, &slave, &slave_name, null, null);
        if (result != 0) {
            return PTYError.OpenFailed;
        }

        return PTY{
            .master = master,
            .slave = slave,
            .slave_name = slave_name[0..std.mem.indexOfScalar(u8, &slave_name, 0).?],
        };
    }

    pub fn close(self: *PTY) void {
        posix.close(self.master);
        posix.close(self.slave);
    }

    pub fn setWindowSize(self: *PTY, rows: u16, cols: u16) !void {
        var ws: posix.win_size = .{
            .ws_row = rows,
            .ws_col = cols,
            .ws_xpixel = 0,
            .ws_ypixel = 0,
        };
        const result = posix.fcntl(self.master, posix.T.IOCSWINSZ, @intFromPtr(&ws));
        if (result != 0) {
            return PTYError.SetTermiosFailed;
        }
    }

    pub fn getWindowSize(self: *PTY) ?struct { rows: u16, cols: u16 } {
        var ws: posix.win_size = undefined;
        const result = posix.fcntl(self.master, posix.T.IOCGWINSZ, @intFromPtr(&ws));
        if (result != 0) {
            return null;
        }
        return .{ .rows = ws.ws_row, .cols = ws.ws_col };
    }

    pub fn nonblock(self: *PTY) void {
        const flags = posix.fcntl(self.master, posix.F.GETFL, 0);
        _ = posix.fcntl(self.master, posix.F.SETFL, flags | posix.O.NONBLOCK);
    }

    pub fn makeControllingTty(self: *PTY) !void {
        const result = posix.ioctl(self.slave, posix.T.IOCSCTTY, 0);
        if (result != 0) {
            return PTYError.OpenFailed;
        }
    }
};

pub fn forkWithPTY() (PTYError!struct { pid: posix.pid_t, pty: PTY }) {
    var pty = try PTY.open();
    errdefer pty.close();

    const pid = posix.fork() catch {
        return PTYError.OpenFailed;
    };

    if (pid == 0) {
        posix.close(pty.master);
        try pty.makeContentingTty();

        const result = posix.setsid();
        _ = result;

        return .{ .pid = 0, .pty = pty };
    }

    posix.close(pty.slave);

    return .{ .pid = pid, .pty = pty };
}
