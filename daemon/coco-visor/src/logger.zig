// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Logging utilities for coco-visor.
//! Spec: Structured JSON logs with timestamp, level, component, message, context.

const std = @import("std");

pub const LogLevel = enum(u8) {
    debug = 0,
    info = 1,
    warn = 2,
    err = 3,
};

var current_level: LogLevel = .info;
var log_writer: ?std.io.AnyWriter = null;

pub fn init(level: LogLevel, writer: std.io.AnyWriter) void {
    current_level = level;
    log_writer = writer;
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

    const ts_nanos = std.time.nanoTimestamp();
    const ts_secs = @divTrunc(ts_nanos, std.time.ns_per_s);
    const ts_nanos_rem = @mod(ts_nanos, std.time.ns_per_s);

    var ts_buf: [64]u8 = undefined;
    const ts_str = std.fmt.bufPrint(&ts_buf, "{d}.{d:0>9}Z", .{ ts_secs, ts_nanos_rem }) catch "unknown";

    const level_str = getLevelStr(level);

    var buf: [256]u8 = undefined;
    var fba = std.heap.FixedBufferAllocator.init(&buf);
    const allocator = fba.allocator();

    const message = std.fmt.allocPrint(allocator, fmt, args) catch return;

    var json_buf = std.ArrayList(u8).init(allocator);
    json_buf.writer().print(
        \\{{"timestamp":"{any}","level":"{any}","component":"coco-visor","message":"{any}"}}
    , .{
        ts_str,
        level_str,
        message,
    }) catch return;

    if (log_writer) |w| {
        w.writeAll(json_buf.items) catch {};
        w.writeAll("\n") catch {};
    } else {
        std.debug.print("{s}\n", .{json_buf.items});
    }
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
