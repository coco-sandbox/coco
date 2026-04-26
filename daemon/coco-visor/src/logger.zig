// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Logging utilities for coco-visor.
//! Spec: Structured JSON logs with timestamp, level, component, message, context.

const std = @import("std");
const os = std.os;

pub const LogLevel = enum(u8) {
    debug = 0,
    info = 1,
    warn = 2,
    err = 3,
};

var current_level: LogLevel = .info;

pub fn init(level: LogLevel, writer: anytype) void {
    _ = writer;
    current_level = level;
}

pub fn setLevel(level: LogLevel) void {
    current_level = level;
}

fn getLevelStr(level: LogLevel) []const u8 {
    return switch (level) {
        .debug => "DEBUG",
        .info => "INFO",
        .warn => "WARN",
        .err => "ERROR",
    };
}

pub fn log(level: LogLevel, comptime fmt: []const u8, args: anytype) void {
    if (@intFromEnum(level) < @intFromEnum(current_level)) return;

    var ts: os.linux.timespec = undefined;
    _ = os.linux.clock_gettime(os.linux.clockid_t.REALTIME, &ts);
    const ts_secs: i64 = ts.sec;
    const ts_nanos_rem: i64 = ts.nsec;

    var ts_buf: [32]u8 = undefined;
    const ts_str = std.fmt.bufPrint(&ts_buf, "{d}.{d:0>9}Z", .{ ts_secs, ts_nanos_rem }) catch "unknown";

    const level_str = getLevelStr(level);

    var msg_buf: [128]u8 = undefined;
    const message = std.fmt.bufPrint(&msg_buf, fmt, args) catch "message too long";

    var json_buf: [512]u8 = undefined;
    const json_str = std.fmt.bufPrint(&json_buf, "{{\"timestamp\":\"{s}\",\"level\":\"{s}\",\"component\":\"coco-visor\",\"message\":\"{s}\"}}\n", .{ ts_str, level_str, message }) catch "log line too long";

    std.debug.print("{s}", .{json_str});
}

pub fn debug(comptime fmt: []const u8, args: anytype) void {
    log(.debug, fmt, args);
}

pub fn info(comptime fmt: []const u8, args: anytype) void {
    log(.info, fmt, args);
}

pub fn warn(comptime fmt: []const u8, args: anytype) void {
    log(.warn, fmt, args);
}

pub fn err(comptime fmt: []const u8, args: anytype) void {
    log(.err, fmt, args);
}
