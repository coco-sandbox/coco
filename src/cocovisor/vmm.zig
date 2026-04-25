// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");

pub const VmmError = error{
    NotInitialized,
    AlreadyBooted,
    NotBooted,
    IoError,
};

pub fn init() VmmError!void {
    // Placeholder - real implementation uses Cloud Hypervisor KVM
}

pub fn boot(rootfs: []const u8, memory_mb: u32, vcpus: u32) VmmError!struct { pid: u32, vsock_cid: u32 } {
    _ = rootfs;
    _ = memory_mb;
    _ = vcpus;
    return .{ .pid = 0, .vsock_cid = 0 };
}

pub fn getState() VmmError!struct { state: u32, pid: u32 } {
    return .{ .state = 1, .pid = 0 };
}

pub fn destroy(force: bool) VmmError!void {
    _ = force;
}
