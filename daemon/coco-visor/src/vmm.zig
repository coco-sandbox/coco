// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VMM with direct KVM integration.
//! Handles MicroVM lifecycle: boot, fork, hibernate, pause, resume.

const std = @import("std");
const posix = std.posix;

const vm_mod = @import("vm.zig");
const vsock = @import("vsock.zig");
const fork_mod = @import("fork.zig");
const checkpoint_mod = @import("checkpoint.zig");
const agent_registry = @import("agent_registry.zig");

pub const VMConfig = vm_mod.VmConfig;
pub const SNAPSHOT_DIR = "/var/lib/coco/hibernation";
pub const CHECKPOINT_DIR = "/var/lib/coco/checkpoints";

pub const VMState = enum(u32) {
    created = 0,
    booting = 1,
    running = 2,
    paused = 3,
    hibernated = 4,
    stopping = 5,
    stopped = 6,
    err_state = 7,
};

pub const VMMError = error{
    NotInitialized,
    AlreadyBooted,
    NotBooted,
    InvalidConfig,
    IoError,
    HypervisorError,
    SnapshotFailed,
    RestoreFailed,
    OutOfMemory,
    ForkNotSupported,
    CheckpointNotSupported,
};

pub const BootResult = struct { pid: u32, vsock_cid: u32 };
pub const ForkResult = struct { child_pid: u32, child_vsock_cid: u32 };
pub const SnapshotResult = struct { duration_ms: u32, size_bytes: u64 };
pub const RestoreResult = struct { pid: u32, vsock_cid: u32, duration_ms: u32 };

var global_next_vsock_cid: u32 = 3;
var vms = std.StringHashMap(*VM).init(std.heap.page_allocator);
var global_metrics: Metrics = .{};

pub const VM = struct {
    config: VMConfig,
    state: VMState,
    pid: u32,
    vm_instance: ?*vm_mod.Vm,
    agent_fd: i32,
    memory_ptr: ?[*]u8,
    memory_size: u64,

    pub fn init(config: VMConfig) VM {
        return .{
            .config = config,
            .state = .created,
            .pid = 0,
            .vm_instance = null,
            .agent_fd = -1,
            .memory_ptr = null,
            .memory_size = 0,
        };
    }

    pub fn boot(self: *VM) VMMError!BootResult {
        if (self.state != .created and self.state != .stopped) {
            return VMMError.AlreadyBooted;
        }

        self.state = .booting;

        std.debug.print("[vmm] Booting VM {any} (mem={d}MB, vcpus={d})\n", .{
            self.config.id, self.config.memory_mb, self.config.vcpus,
        });

        const mem_size = @as(u64, self.config.memory_mb) * 1024 * 1024;
        const mem_slice = posix.mmap(
            null,
            mem_size,
            posix.PROT.READ | posix.PROT.WRITE,
            posix.MAP{ .TYPE = .SHARED, .ANONYMOUS = true },
            -1,
            0,
        ) catch {
            self.state = .err_state;
            return VMMError.OutOfMemory;
        };
        self.memory_ptr = mem_slice.ptr;
        self.memory_size = mem_size;

        var vm = vm_mod.Vm.create(&self.config, self.memory_ptr.?, mem_size, std.heap.page_allocator) catch |e| {
            std.debug.print("[vmm] Failed to create VM: {}\n", .{e});
            self.state = .err_state;
            return VMMError.HypervisorError;
        };

        _ = vm.start() catch |e| {
            std.debug.print("[vmm] Failed to start vCPU: {}\n", .{e});
            vm.destroy();
            self.state = .err_state;
            return VMMError.HypervisorError;
        };

        self.pid = @intCast(std.os.linux.getpid());

        var agent_fd: i32 = -1;
        var retry: u32 = 0;
        while (retry < 30) : (retry += 1) {
            if (agent_registry.get(self.config.vsock_cid)) |fd| {
                agent_fd = fd;
                break;
            }
            std.time.sleep(200 * std.time.ns_per_ms);
        }

        if (agent_fd < 0) {
            std.debug.print("[vmm] Failed to get agent from registry after 30 retries\n", .{});
            vm.destroy();
            self.state = .err_state;
            return VMMError.HypervisorError;
        }

        const vm_ptr = std.heap.page_allocator.create(vm_mod.Vm) catch {
            posix.close(agent_fd);
            vm.destroy();
            return VMMError.OutOfMemory;
        };
        vm_ptr.* = vm;

        self.vm_instance = vm_ptr;
        self.agent_fd = agent_fd;
        self.state = .running;

        std.debug.print("[vmm] VM {any} booted: pid={d}, cid={d}\n", .{
            self.config.id, self.pid, self.config.vsock_cid,
        });

        registerVM(self.config.id, self);

        return .{ .pid = self.pid, .vsock_cid = self.config.vsock_cid };
    }

    pub fn pause(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        if (self.vm_instance) |vm| {
            vm.pause();
        }
        self.state = .paused;
        std.debug.print("[vmm] VM {any} paused\n", .{self.config.id});
    }

    pub fn resume_(self: *VM) VMMError!void {
        if (self.state != .paused) {
            return VMMError.NotBooted;
        }

        if (self.vm_instance) |vm| {
            vm.resume_();
        }
        self.state = .running;
        std.debug.print("[vmm] VM {any} resumed\n", .{self.config.id});
    }

    pub fn destroy(self: *VM) VMMError!void {
        if (self.state == .stopped) {
            return;
        }

        self.state = .stopping;
        std.debug.print("[vmm] Destroying VM {any}\n", .{self.config.id});

        if (self.agent_fd >= 0) {
            posix.close(self.agent_fd);
            self.agent_fd = -1;
        }

        if (self.vm_instance) |vm| {
            vm.destroy();
            std.heap.page_allocator.destroy(vm);
            self.vm_instance = null;
        }

        if (self.memory_ptr) |ptr| {
            const aligned_ptr = @as([*]align(std.heap.page_size_min) u8, @alignCast(ptr));
            posix.munmap(aligned_ptr[0..self.memory_size]);
            self.memory_ptr = null;
        }

        self.state = .stopped;
        self.pid = 0;

        unregisterVM(self.config.id);

        std.debug.print("[vmm] VM {any} destroyed\n", .{self.config.id});
    }

    pub fn fork(self: *VM) VMMError!ForkResult {
        if (self.state != .running and self.state != .paused) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Forking VM {any} (using btrfs reflinks)\n", .{self.config.id});

        if (self.state == .running) {
            self.pause() catch {};
        }

        const child_id = std.fmt.allocPrint(
            std.heap.page_allocator,
            "{any}-fork-{d}",
            .{ self.config.id, std.time.timestamp() },
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(child_id);

        const child_cid = allocateVsockCid();

        const child_config = VMConfig{
            .id = try std.heap.page_allocator.dupe(u8, child_id),
            .rootfs = self.config.rootfs,
            .kernel = self.config.kernel,
            .initrd = self.config.initrd,
            .memory_mb = self.config.memory_mb,
            .vcpus = self.config.vcpus,
            .vsock_cid = child_cid,
            .tap_name = self.config.tap_name,
        };

        var fork_manager = fork_mod.ForkManager.init(std.heap.page_allocator, "/var/lib/coco/templates");
        fork_manager.createFork(self.config.id, child_id);

        const child_mem_size = self.memory_size;
        const child_mem = posix.mmap(
            null,
            child_mem_size,
            posix.PROT.READ | posix.PROT.WRITE,
            posix.MAP{ .TYPE = .SHARED, .ANONYMOUS = true },
            -1,
            0,
        ) catch {
            self.resume_() catch {};
            return VMMError.OutOfMemory;
        };
        @memcpy(child_mem[0..child_mem_size], self.memory_ptr.?[0..child_mem_size]);

        var child_vm = vm_mod.Vm.create(&child_config, child_mem.ptr, child_mem_size, std.heap.page_allocator) catch |e| {
            std.debug.print("[vmm] Fork VM create failed: {}\n", .{e});
            self.resume_() catch {};
            return VMMError.HypervisorError;
        };

        _ = child_vm.start() catch |e| {
            std.debug.print("[vmm] Fork vCPU start failed: {}\n", .{e});
            child_vm.destroy();
            self.resume_() catch {};
            return VMMError.HypervisorError;
        };

        var child_agent_fd: i32 = -1;
        var retry: u32 = 0;
        while (retry < 30) : (retry += 1) {
            if (agent_registry.get(child_cid)) |fd| {
                child_agent_fd = fd;
                break;
            }
            std.time.sleep(200 * std.time.ns_per_ms);
        }

        if (child_agent_fd < 0) {
            child_vm.destroy();
            self.resume_() catch {};
            return VMMError.HypervisorError;
        }

        const child_vm_ptr = std.heap.page_allocator.create(VM) catch {
            posix.close(child_agent_fd);
            child_vm.destroy();
            self.resume_() catch {};
            return VMMError.OutOfMemory;
        };

        const child_pid: u32 = @intCast(std.os.linux.getpid());
        child_vm_ptr.* = VM.init(child_config);
        child_vm_ptr.state = .running;
        child_vm_ptr.pid = child_pid;
        child_vm_ptr.vm_instance = std.heap.page_allocator.create(vm_mod.Vm) catch {
            posix.close(child_agent_fd);
            child_vm.destroy();
            std.heap.page_allocator.destroy(child_vm_ptr);
            self.resume_() catch {};
            return VMMError.OutOfMemory;
        };
        child_vm_ptr.vm_instance.?.* = child_vm;
        child_vm_ptr.agent_fd = child_agent_fd;
        child_vm_ptr.memory_ptr = child_mem.ptr;
        child_vm_ptr.memory_size = child_mem_size;

        registerVM(child_id, child_vm_ptr);

        self.resume_() catch {};

        std.debug.print("[vmm] Fork complete: child_id={any}, child_pid={d}, child_cid={d}\n", .{
            child_id, child_pid, child_cid,
        });

        return .{ .child_pid = child_pid, .child_vsock_cid = child_cid };
    }

    pub fn hibernate(self: *VM) VMMError!SnapshotResult {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        const start = std.time.nanoTimestamp();

        std.debug.print("[vmm] Hibernate VM {any}\n", .{self.config.id});

        std.fs.makeDirAbsolute(SNAPSHOT_DIR) catch {};

        const snap_dir = std.fmt.allocPrint(
            std.heap.page_allocator,
            "{any}/{any}",
            .{ SNAPSHOT_DIR, self.config.id },
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(snap_dir);
        std.fs.makeDirAbsolute(snap_dir) catch {};

        if (self.vm_instance) |vm| {
            vm.pause();
        }
        self.state = .hibernated;

        if (self.memory_ptr) |mem_ptr| {
            var checkpoint = checkpoint_mod.CheckpointManager.init(std.heap.page_allocator);
            const meta = checkpoint_mod.CheckpointMetadata{
                .id = self.config.id,
                .memory_size = self.memory_size,
                .compressed_size = 0,
                .timestamp = std.time.timestamp(),
                .memory_mb = self.config.memory_mb,
                .vcpus = self.config.vcpus,
                .kernel = self.config.kernel,
                .rootfs = self.config.rootfs,
            };
            _ = checkpoint.createCheckpoint(self.config.id, mem_ptr, self.memory_size, meta);
        }

        const duration: i128 = std.time.nanoTimestamp() - start;
        const duration_ms = @as(u32, @intCast(@divTrunc(duration, 1_000_000)));

        global_metrics.recordHibernate(@intCast(duration));

        std.debug.print("[vmm] Hibernate complete: {d}ms\n", .{duration_ms});

        return .{ .duration_ms = duration_ms, .size_bytes = self.memory_size };
    }

    pub fn resumeFromHibernate(self: *VM) VMMError!BootResult {
        if (self.state != .hibernated) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Resuming VM {any} from hibernate\n", .{self.config.id});

        if (self.vm_instance) |vm| {
            vm.resume_();
        }
        self.state = .running;

        std.debug.print("[vmm] Resume from hibernate complete\n", .{});

        return .{ .pid = self.pid, .vsock_cid = self.config.vsock_cid };
    }

    pub fn snapshot(self: *VM, incremental: bool) VMMError!SnapshotResult {
        _ = incremental;
        if (self.state != .running and self.state != .paused) {
            return VMMError.NotBooted;
        }

        const start = std.time.nanoTimestamp();

        if (self.state == .running) {
            try self.pause();
        }

        if (self.memory_ptr) |mem_ptr| {
            var checkpoint = checkpoint_mod.CheckpointManager.init(std.heap.page_allocator);
            const meta = checkpoint_mod.CheckpointMetadata{
                .id = self.config.id,
                .memory_size = self.memory_size,
                .compressed_size = 0,
                .timestamp = std.time.timestamp(),
                .memory_mb = self.config.memory_mb,
                .vcpus = self.config.vcpus,
                .kernel = self.config.kernel,
                .rootfs = self.config.rootfs,
            };
            const size = checkpoint.createCheckpoint(self.config.id, mem_ptr, self.memory_size, meta) catch |e| {
                std.debug.print("[vmm] Snapshot failed: {}\n", .{e});
                return VMMError.SnapshotFailed;
            };
            _ = size;
        }

        try self.resume_();

        const duration = @as(u64, std.time.nanoTimestamp() - start);
        const duration_ms = @as(u32, @intCast(@divTrunc(duration, 1_000_000)));

        std.debug.print("[vmm] Snapshot complete: {d}ms\n", .{duration_ms});

        return .{ .duration_ms = duration_ms, .size_bytes = self.memory_size };
    }
};

pub fn allocateVsockCid() u32 {
    _ = @atomicRmw(u32, &global_next_vsock_cid, .Add, 1, .seq_cst);
    return global_next_vsock_cid - 1;
}

pub fn getVMs() *std.StringHashMap(*VM) {
    return &vms;
}

pub fn registerVM(id: []const u8, vm: *VM) void {
    vms.put(id, vm) catch {};
}

pub fn unregisterVM(id: []const u8) void {
    _ = vms.remove(id);
}

pub fn getOrCreateVM(config: VMConfig) VMMError!*VM {
    if (vms.get(config.id)) |existing| {
        return existing;
    }

    const vm = std.heap.page_allocator.create(VM) catch return VMMError.OutOfMemory;
    vm.* = VM.init(config);
    try vms.put(config.id, vm);
    return vm;
}

pub fn removeVM(id: []const u8) void {
    if (vms.get(id)) |vm| {
        vm.destroy() catch {};
    }
    _ = vms.remove(id);
}

pub const Metrics = struct {
    boot_count: u32 = 0,
    fork_count: u32 = 0,
    hibernate_count: u32 = 0,
    resume_count: u32 = 0,
    pause_count: u32 = 0,
    total_boot_time_ns: u64 = 0,
    total_fork_time_ns: u64 = 0,
    total_hibernate_time_ns: u64 = 0,

    pub fn recordBoot(self: *Metrics, duration_ns: u64) void {
        self.boot_count += 1;
        self.total_boot_time_ns += duration_ns;
    }

    pub fn avgBootTimeNs(self: *Metrics) u64 {
        if (self.boot_count == 0) return 0;
        return self.total_boot_time_ns / self.boot_count;
    }

    pub fn recordFork(self: *Metrics, duration_ns: u64) void {
        self.fork_count += 1;
        self.total_fork_time_ns += duration_ns;
    }

    pub fn recordHibernate(self: *Metrics, duration_ns: u64) void {
        self.hibernate_count += 1;
        self.total_hibernate_time_ns += duration_ns;
    }
};

pub fn getMetrics() *Metrics {
    return &global_metrics;
}
