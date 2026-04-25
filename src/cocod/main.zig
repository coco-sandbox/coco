// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");

pub fn main() !void {
    try std.io.getStdOut().writeAll("CoCod guest agent (placeholder)\n");
    try std.io.getStdOut().writeAll("Runs inside MicroVM, handles exec via vsock\n");
}