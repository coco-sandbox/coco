// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VM implementation with direct KVM.

const std = @import("std");
const posix = std.posix;
const linux = std.os.linux;
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
    vcpu_instances: []vcpu.VCpu,
    vcpu_threads: []?std.Thread,
    vsock_fd: i32,
    tap_fd: i32,
    cid: u32,

    pub fn create(config: *const VmConfig, mem_ptr: [*]u8, mem_size: u64, allocator: std.mem.Allocator) !Vm {
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

        const cmdline = if (config.tap_name.len > 0)
            try std.fmt.allocPrint(allocator, "console=ttyS0 reboot=k panic=1 pci=off netdev=eth0,${}", .{config.tap_name})
        else
            "console=ttyS0 reboot=k panic=1 pci=off";
        defer if (config.tap_name.len > 0) allocator.free(cmdline);

        try memory.setupBootParams(&mem, cmdline, @intCast(mem_size / 1024 / 1024), initrd_info.addr, initrd_info.size);

        memory.setupGdt(&mem);

        const mmap_size = try kvm_mod.getVcpuMmapSize(kvm_fd);

        const vcpu_count = @min(config.vcpus, 16);
        var vcpu_instances = try allocator.alloc(vcpu.VCpu, vcpu_count);
        errdefer allocator.free(vcpu_instances);

        var i: u32 = 0;
        while (i < vcpu_count) : (i += 1) {
            vcpu_instances[i] = try vcpu.VCpu.create(vm_fd, i, mmap_size);
            if (i == 0) {
                try vcpu_instances[i].setup32(memory.KERNEL_LOAD_ADDR, 0x7000);
            }
        }

        const vsock_fd = try vsock.openAndAssignCid(config.vsock_cid);
        errdefer posix.close(vsock_fd);

        var tap_fd: i32 = -1;
        if (config.tap_name.len > 0) {
            tap_fd = createTapDevice(config.tap_name) catch -1;
        }

        return Vm{
            .kvm_fd = kvm_fd,
            .vm_fd = vm_fd,
            .mem = mem,
            .vcpu_instances = vcpu_instances,
            .vcpu_threads = try allocator.alloc(?std.Thread, vcpu_count),
            .vsock_fd = vsock_fd,
            .tap_fd = tap_fd,
            .cid = config.vsock_cid,
        };
    }

    fn createTapDevice(name: []const u8) !i32 {
        const TUNSETIFF: u32 = 0x400454ca;
        const IFF_TAP: u32 = 0x0002;
        const IFF_NO_PI: u32 = 0x1000;

        const fd = try posix.open("/dev/net/tun", .{ .ACCMODE = .RDWR }, 0);

        var ifr: [256]u8 = undefined;
        @memset(&ifr, 0);

        const name_len = @min(name.len, 15);
        @memcpy(ifr[0..name_len], name[0..name_len]);

        const flags: u16 = @truncate(IFF_TAP | IFF_NO_PI);
        std.mem.writeInt(u16, ifr[16..18], flags, .little);

        const result = linux.ioctl(fd, TUNSETIFF, @intFromPtr(&ifr));
        if (result < 0) {
            posix.close(fd);
            return error.TapCreationFailed;
        }

        return fd;
    }

    pub fn start(self: *Vm) !std.Thread {
        var threads: usize = 0;
        for (self.vcpu_instances) |*v| {
            self.vcpu_threads[threads] = try std.Thread.spawn(.{}, vcpuRunner, .{v});
            threads += 1;
        }

        if (self.tap_fd >= 0) {
            try setUpNetwork();
        }

        return self.vcpu_threads[0] orelse unreachable;
    }

    fn setUpNetwork() !void {
        // Network setup would be done here
    }

    fn vcpuRunner(v: *vcpu.VCpu) void {
        v.runLoop() catch {};
    }

    pub fn pause(self: *Vm) void {
        for (self.vcpu_instances) |*v| {
            v.halt.store(true, .seq_cst);
        }
    }

    pub fn resume_(self: *Vm) void {
        for (self.vcpu_instances) |*v| {
            v.halt.store(false, .seq_cst);
        }
    }

    pub fn destroy(self: *Vm) void {
        for (self.vcpu_instances) |*v| {
            v.halt.store(true, .seq_cst);
        }

        for (self.vcpu_threads) |thread| {
            if (thread) |t| {
                t.join();
            }
        }

        for (self.vcpu_instances) |*v| {
            v.deinit();
        }

        self.mem.deinit();
        posix.close(self.vm_fd);
        posix.close(self.kvm_fd);

        if (self.vsock_fd >= 0) {
            posix.close(self.vsock_fd);
        }

        if (self.tap_fd >= 0) {
            posix.close(self.tap_fd);
        }
    }
};
