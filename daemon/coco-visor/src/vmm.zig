// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VMM with direct KVM integration.
//! Handles MicroVM lifecycle: boot, fork, hibernate, pause, resume.

const std = @import("std");

const clh = @import("clh.zig");
const vm_mod = @import("vm.zig");
const vsock = @import("vsock.zig");
pub const VMConfig = clh.VMConfig;

// =============================================================================
// Constants
// =============================================================================

const SNAPSHOT_DIR = "/var/lib/coco/hibernation";
const CLH_API_SOCKET_DIR = "/run/coco/vm/";

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
};

pub const BootResult = struct { pid: u32, vsock_cid: u32 };

// =============================================================================
// VM Config
// =============================================================================

// VMConfig is imported from clh.zig

// =============================================================================
// VM Instance
// =============================================================================

pub const VM = struct {
    config: VMConfig,
    state: VMState,
    pid: u32,
    vm_instance: ?*vm_mod.Vm,
    agent_fd: i32,

    pub fn init(config: VMConfig) VM {
        return .{
            .config = config,
            .state = .created,
            .pid = 0,
            .vm_instance = null,
            .agent_fd = -1,
        };
    }

    pub fn boot(self: *VM) VMMError!BootResult {
        if (self.state != .created and self.state != .stopped) {
            return VMMError.AlreadyBooted;
        }

        self.state = .booting;

        std.debug.print("[vmm] Booting VM {s} (mem={d}MB, vcpus={d})\n", .{
            self.config.id, self.config.memory_mb, self.config.vcpus,
        });

        var vm = vm_mod.Vm.create(&self.config, std.heap.page_allocator) catch |e| {
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

        var retry_count: u32 = 0;
        const max_retries = 30;
        var agent_fd: i32 = -1;

        while (retry_count < max_retries) {
            agent_fd = vsock.connectToAgent(self.config.vsock_cid, 4747) catch {
                retry_count += 1;
                if (retry_count >= max_retries) break;
                std.time.sleep(200 * std.time.ns_per_ms);
                continue;
            };
            break;
        }

        if (agent_fd < 0) {
            std.debug.print("[vmm] Failed to connect to agent after {d} retries\n", .{max_retries});
            vm.destroy();
            self.state = .err_state;
            return VMMError.HypervisorError;
        }

        const vm_ptr = std.heap.page_allocator.create(vm_mod.Vm) catch {
            std.posix.close(agent_fd);
            vm.destroy();
            return VMMError.OutOfMemory;
        };
        vm_ptr.* = vm;

        self.vm_instance = vm_ptr;
        self.agent_fd = agent_fd;
        self.state = .running;

        std.debug.print("[vmm] VM {s} booted: pid={d}, cid={d}\n", .{
            self.config.id, self.pid, self.config.vsock_cid,
        });

        registerVM(self.config.id, self);

        return .{ .pid = self.pid, .vsock_cid = self.config.vsock_cid };
    }

    pub fn pause(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        var vm = self.vm_instance orelse return VMMError.NotBooted;
        vm.pause();
        self.state = .paused;
        std.debug.print("[vmm] VM {s} paused\n", .{self.config.id});
    }

    pub fn resume_(self: *VM) VMMError!void {
        if (self.state != .paused) {
            return VMMError.NotBooted;
        }

        var vm = self.vm_instance orelse return VMMError.NotBooted;
        vm.resume_();
        self.state = .running;
        std.debug.print("[vmm] VM {s} resumed\n", .{self.config.id});
    }

    pub fn destroy(self: *VM) VMMError!void {
        if (self.state == .stopped) {
            return;
        }

        self.state = .stopping;
        std.debug.print("[vmm] Destroying VM {s}\n", .{self.config.id});

        if (self.agent_fd >= 0) {
            std.posix.close(self.agent_fd);
            self.agent_fd = -1;
        }

        if (self.vm_instance) |vm| {
            vm.destroy();
            std.heap.page_allocator.destroy(vm);
            self.vm_instance = null;
        }

        self.state = .stopped;
        self.pid = 0;

        unregisterVM(self.config.id);

        std.debug.print("[vmm] VM {s} destroyed\n", .{self.config.id});
    }

    pub fn fork(self: *VM) VMMError!struct { child_pid: u32, child_vsock_cid: u32 } {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Forking VM {s}\n", .{self.config.id});

        self.pause() catch {};

        const child_id = std.fmt.allocPrint(
            std.heap.page_allocator,
            "{s}-fork-{d}",
            .{ self.config.id, std.time.timestamp() },
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(child_id);

        const child_cid = allocateVsockCid();

        const child_config = VMConfig{
            .id = child_id,
            .rootfs = self.config.rootfs,
            .kernel = self.config.kernel,
            .initrd = self.config.initrd,
            .memory_mb = self.config.memory_mb,
            .vcpus = self.config.vcpus,
            .vsock_cid = child_cid,
            .tap_name = "",
        };

        var child_vm = vm_mod.Vm.create(&child_config, std.heap.page_allocator) catch |e| {
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
        var retry_count: u32 = 0;
        while (retry_count < 30) {
            child_agent_fd = vsock.connectToAgent(child_cid, 4747) catch {
                retry_count += 1;
                std.time.sleep(200 * std.time.ns_per_ms);
                continue;
            };
            break;
        }

        if (child_agent_fd < 0) {
            child_vm.destroy();
            self.resume_() catch {};
            return VMMError.HypervisorError;
        }

        const child_vm_ptr = std.heap.page_allocator.create(VM) catch {
            std.posix.close(child_agent_fd);
            child_vm.destroy();
            self.resume_() catch {};
            return VMMError.OutOfMemory;
        };

        const child_pid: u32 = @intCast(std.os.linux.getpid());
        child_vm_ptr.* = VM.init(child_config);
        child_vm_ptr.state = .running;
        child_vm_ptr.pid = child_pid;
        child_vm_ptr.vm_instance = std.heap.page_allocator.create(vm_mod.Vm) catch {
            std.posix.close(child_agent_fd);
            child_vm.destroy();
            std.heap.page_allocator.destroy(child_vm_ptr);
            self.resume_() catch {};
            return VMMError.OutOfMemory;
        };
        child_vm_ptr.vm_instance.?.* = child_vm;
        child_vm_ptr.agent_fd = child_agent_fd;

        registerVM(child_id, child_vm_ptr);

        self.resume_() catch {};

        std.debug.print("[vmm] Fork complete: child_id={s}, child_pid={d}, child_cid={d}\n", .{
            child_id, child_pid, child_cid,
        });

        return .{ .child_pid = child_pid, .child_vsock_cid = child_cid };
    }

    pub fn hibernate(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Hibernate VM {s}\n", .{self.config.id});

        std.fs.makeDirAbsolute(SNAPSHOT_DIR) catch {};

        const snap_dir = std.fmt.allocPrint(
            std.heap.page_allocator,
            "{s}/{s}",
            .{ SNAPSHOT_DIR, self.config.id },
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(snap_dir);
        std.fs.makeDirAbsolute(snap_dir) catch {};

        if (self.vm_instance) |vm| {
            vm.pause();
        }
        self.state = .hibernated;

        std.debug.print("[vmm] Hibernate complete: {s}\n", .{snap_dir});
    }

    pub fn resumeFromHibernate(self: *VM) VMMError!void {
        if (self.state != .hibernated) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Resuming VM {s} from hibernate\n", .{self.config.id});

        if (self.vm_instance) |vm| {
            vm.resume_();
        }
        self.state = .running;

        std.debug.print("[vmm] Resume from hibernate complete\n", .{});
    }
};

// =============================================================================
// Helper Functions
// =============================================================================

fn cleanDirectory(dir_path: []const u8) void {
    std.fs.deleteFileAbsolute(dir_path) catch {};
}

fn allocateVsockCid() u32 {
    // Simple counter, in production would need coordination
    _ = @atomicRmw(u32, &global_next_vsock_cid, .Add, 1, .seq_cst);
    return global_next_vsock_cid - 1;
}

var global_next_vsock_cid: u32 = 3;

// =============================================================================
// Global VM Registry
// =============================================================================

var vms = std.StringHashMap(*VM).init(std.heap.page_allocator);

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

// =============================================================================
// Sandbox Metrics (for performance tracking)
// =============================================================================

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

var global_metrics: Metrics = .{};

pub fn getMetrics() *Metrics {
    return &global_metrics;
}
