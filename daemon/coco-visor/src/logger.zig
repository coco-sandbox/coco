// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Logging utilities for coco-visor.

const std = @import("std");

pub const LogLevel = enum(u8) {
    debug = 0,
    info = 1,
    warn = 2,
    error = 3,
};

pub fn log(level: LogLevel, comptime fmt: []const u8, args: anytype) void {
    std.debug.print("[coco-visor] " ++ fmt ++ "\n", args);
    _ = level;
}
