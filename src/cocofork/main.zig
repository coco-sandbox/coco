// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");

pub fn main() !void {
    try std.io.getStdOut().writeAll("CocoFork daemon (placeholder)\n");
    try std.io.getStdOut().writeAll("Instant VM forking via memory snapshot + CoW\n");
}
