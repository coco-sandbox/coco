// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Configuration for coco-visor.

const std = @import("std");

pub const Config = struct {
    socket_path: []const u8 = "/run/coco/visor.sock",
    memory_mb: u32 = 512,
    vcpus: u32 = 2,
    kernel_path: []const u8 = "/var/lib/coco/vmlinux",
    initrd_path: []const u8 = "",
    vsock_port: u32 = 4747,
};

pub fn loadConfig() Config {
    return Config{};
}
