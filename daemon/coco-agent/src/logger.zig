// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Structured JSON logger for cocod (guest agent).

const std = @import("std");

pub const Level = enum(u8) {
    debug = 0,
    info = 1,
    warn = 2,
    err = 3,

    pub fn toString(self: Level) []const u8 {
        return switch (self) {
            .debug => "DEBUG",
            .info => "INFO",
            .warn => "WARN",
            .err => "ERROR",
        };
    }
};

pub const Record = struct {
    timestamp: []const u8,
    level: []const u8,
    component: []const u8,
    message: []const u8,
    trace_id: ?[]const u8 = null,
    sandbox_id: ?[]const u8 = null,

    pub fn writeJSON(self: Record, writer: anytype) !void {
        try writer.print("{{\"timestamp\":\"{s}\",\"level\":\"{s}\",\"component\":\"{s}\",\"message\":\"", .{
            self.timestamp,
            self.level,
            self.component,
        });

        for (self.message) |c| {
            switch (c) {
                '"' => try writer.writeAll("\\\""),
                '\\' => try writer.writeAll("\\\\"),
                '\n' => try writer.writeAll("\\n"),
                '\r' => try writer.writeAll("\\r"),
                '\t' => try writer.writeAll("\\t"),
                else => try writer.writeByte(c),
            }
        }

        try writer.writeAll("\"");

        if (self.trace_id) |tid| {
            try writer.print(",\"trace_id\":\"{s}\"", .{tid});
        }

        if (self.sandbox_id) |sid| {
            try writer.print(",\"sandbox_id\":\"{s}\"", .{sid});
        }

        try writer.writeAll("}\n");
    }
};

pub const Logger = struct {
    level: Level,
    component: []const u8,
    writer: std.io.Writer(std.fs.File, std.posix.WriteError, std.fs.File.write),

    pub fn init(component: []const u8) Logger {
        return .{
            .level = .info,
            .component = component,
            .writer = std.io.getStdOut().writer(),
        };
    }

    pub fn setLevel(logger: *Logger, level: Level) void {
        logger.level = level;
    }

    fn log(logger: Logger, level: Level, msg: []const u8, trace_id: ?[]const u8, sandbox_id: ?[]const u8) void {
        if (@intFromEnum(level) < @intFromEnum(logger.level)) {
            return;
        }

        var timestamp: [64]u8 = undefined;
        const ts = std.fmt.fmtIntToSlice(timestamp[0..], @as(u64, @intCast(std.time.timestamp())), 10);

        const record = Record{
            .timestamp = ts,
            .level = level.toString(),
            .component = logger.component,
            .message = msg,
            .trace_id = trace_id,
            .sandbox_id = sandbox_id,
        };

        record.writeJSON(logger.writer) catch return;
    }

    pub fn debug(logger: Logger, msg: []const u8) void {
        logger.log(.debug, msg, null, null);
    }

    pub fn info(logger: Logger, msg: []const u8) void {
        logger.log(.info, msg, null, null);
    }

    pub fn warn(logger: Logger, msg: []const u8) void {
        logger.log(.warn, msg, null, null);
    }

    pub fn err(logger: Logger, msg: []const u8) void {
        logger.log(.err, msg, null, null);
    }
};

var global_logger = Logger.init("cocod");

pub fn getLogger() *Logger {
    return &global_logger;
}

pub fn debug(msg: []const u8) void {
    global_logger.log(.debug, msg, null, null);
}
pub fn info(msg: []const u8) void {
    global_logger.log(.info, msg, null, null);
}
pub fn warn(msg: []const u8) void {
    global_logger.log(.warn, msg, null, null);
}
pub fn err(msg: []const u8) void {
    global_logger.log(.err, msg, null, null);
}
