// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VM implementation with direct KVM.

const std = @import("std");
const posix = std.posix;
const kvm_mod = @import("kvm.zig");
const memory = @import("memory.zig");
const vcpu = @import("vcpu.zig");
const vsock = @import("vsock.zig");

pub const VmConfig = struct {
    id: []const u8,
    rootfs: []const u8,
    kernel: []const u8 = "/var/lib/coco/vmlinux",
    initrd: []const u8 = "",
    memory_mb: u32 = 512,
    vcpus: u32 = 2,
    vsock_cid: u32,
    tap_name: []const u8 = "",
};

pub const Vm = struct {
    kvm_fd: i32,
    vm_fd: i32,
    mem: memory.GuestMemory,
    vcpu_instance: vcpu.VCpu,
    vcpu_thread: ?std.Thread,
    vsock_fd: i32,
    cid: u32,

    pub fn create(config: *const VmConfig, mem_ptr: [*]u8, mem_size: u64, allocator: std.mem.Allocator) !Vm {
        _ = allocator;

        const kvm_fd = try kvm_mod.open();
        errdefer posix.close(kvm_fd);

        const vm_fd = try kvm_mod.createVm(kvm_fd);
        errdefer posix.close(vm_fd);

        var mem = memory.GuestMemory{
            .ptr = mem_ptr,
            .size = mem_size,
        };

        try kvm_mod.setUserMemoryRegion(vm_fd, &.{
            .slot = 0,
            .flags = 0,
            .guest_phys_addr = 0,
            .memory_size = mem_size,
            .userspace_addr = @intFromPtr(mem_ptr),
        });

        _ = try memory.loadKernel(&mem, config.kernel);

        const initrd_info = try memory.loadInitrd(&mem, config.initrd);

        try memory.setupBootParams(&mem, "console=ttyS0 reboot=k panic=1 pci=off", @intCast(mem_size / 1024 / 1024), initrd_info.addr, initrd_info.size);

        memory.setupGdt(&mem);

        const mmap_size = try kvm_mod.getVcpuMmapSize(kvm_fd);

        var v = try vcpu.VCpu.create(vm_fd, 0, mmap_size);
        errdefer v.deinit();

        try v.setup32(memory.KERNEL_LOAD_ADDR, 0x7000);

        const vsock_fd = try vsock.openAndAssignCid(config.vsock_cid);
        errdefer posix.close(vsock_fd);

        return Vm{
            .kvm_fd = kvm_fd,
            .vm_fd = vm_fd,
            .mem = mem,
            .vcpu_instance = v,
            .vcpu_thread = null,
            .vsock_fd = vsock_fd,
            .cid = config.vsock_cid,
        };
    }

    pub fn start(self: *Vm) !std.Thread {
        const thread = try std.Thread.spawn(.{}, vcpuRunner, .{self});
        self.vcpu_thread = thread;
        return thread;
    }

    fn vcpuRunner(self: *Vm) void {
        self.vcpu_instance.runLoop() catch {};
    }

    pub fn pause(self: *Vm) void {
        self.vcpu_instance.halt.store(true, .seq_cst);
    }

    pub fn resume_(self: *Vm) void {
        self.vcpu_instance.halt.store(false, .seq_cst);
    }

    pub fn destroy(self: *Vm) void {
        self.vcpu_instance.halt.store(true, .seq_cst);
        if (self.vcpu_thread) |thread| {
            thread.join();
        }

        self.vcpu_instance.deinit();
        self.mem.deinit();
        posix.close(self.vm_fd);
        posix.close(self.kvm_fd);
        if (self.vsock_fd >= 0) {
            posix.close(self.vsock_fd);
        }
    }
};
