// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");

const SOCK_PATH = "/run/coco/net.sock";

pub fn main() !void {
    try std.io.getStdOut().writeAll("CocoNet daemon (placeholder)\n");
    try std.io.getStdOut().writeAll("eBPF-powered networking for Coco Sandbox\n");
}