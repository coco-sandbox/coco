// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VMM tests for Cloud Hypervisor integration.

const std = @import("std");
const vmm = @import("vmm");

test "VM init creates VM with correct config" {
    const config = vmm.VMConfig{
        .id = "test-vm",
        .rootfs = "/var/lib/coco/images/test.rootfs",
        .kernel = "/var/lib/coco/vmlinux",
        .initrd = "",
        .memory_mb = 512,
        .vcpus = 2,
        .vsock_cid = 3,
        .tap_name = "",
    };

    var vm = vmm.VM.init(config);
    try std.testing.expect(vm.config.memory_mb == 512);
    try std.testing.expect(vm.state == .created);
}
