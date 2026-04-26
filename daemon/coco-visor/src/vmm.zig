// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VMM integration with Cloud Hypervisor.
//! Handles MicroVM lifecycle: boot, fork, hibernate, pause, resume.

const std = @import("std");

const clh = @import("clh.zig");
const CLHClient = clh.CLHClient;
const CLHError = clh.CLHError;
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
    clh_client: ?CLHClient,

    pub fn init(config: VMConfig) VM {
        return .{
            .config = config,
            .state = .created,
            .pid = 0,
            .clh_client = null,
        };
    }

    pub fn boot(self: *VM) VMMError!BootResult {
        if (self.state != .created and self.state != .stopped) {
            return VMMError.AlreadyBooted;
        }

        self.state = .booting;

        // Create VM run directory
        const run_dir = std.fmt.allocPrint(
            std.heap.page_allocator,
            "/run/coco/vm/{s}",
            .{self.config.id},
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(run_dir);
        std.fs.makeDirAbsolute(run_dir) catch {};

        // Create VM lib directory
        const lib_dir = std.fmt.allocPrint(
            std.heap.page_allocator,
            "/var/lib/coco/vm/{s}",
            .{self.config.id},
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(lib_dir);
        std.fs.makeDirAbsolute(lib_dir) catch {};

        std.debug.print("[vmm] Booting VM {s} (mem={d}MB, vcpus={d})\n", .{
            self.config.id, self.config.memory_mb, self.config.vcpus,
        });

        // Spawn Cloud Hypervisor process
        self.clh_client = CLHClient.spawn(self.config.id, &self.config, std.heap.page_allocator) catch |e| {
            std.debug.print("[vmm] Failed to spawn CLH: {}\n", .{e});
            self.state = .err_state;
            return VMMError.HypervisorError;
        };

        var clh_client = self.clh_client.?;
        const result = clh_client.boot(&self.config) catch |e| {
            std.debug.print("[vmm] CLH boot command failed: {}\n", .{e});
            self.state = .err_state;
            return VMMError.HypervisorError;
        };

        self.pid = result.pid;
        self.state = .running;

        std.debug.print("[vmm] VM {s} booted: pid={d}, cid={d}\n", .{
            self.config.id, self.pid, self.config.vsock_cid,
        });

        // Register in global VM registry
        registerVM(self.config.id, self);

        return .{ .pid = result.pid, .vsock_cid = result.vsock_cid };
    }

    pub fn pause(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        var clh_client = self.clh_client orelse return VMMError.NotBooted;
        clh_client.pause(self.config.id) catch |e| {
            std.debug.print("[vmm] Pause failed: {}\n", .{e});
            return VMMError.HypervisorError;
        };

        self.state = .paused;
        std.debug.print("[vmm] VM {s} paused\n", .{self.config.id});
    }

    pub fn resume_(self: *VM) VMMError!void {
        if (self.state != .paused) {
            return VMMError.NotBooted;
        }

        var clh_client = self.clh_client orelse return VMMError.NotBooted;
        clh_client.resumeVm(self.config.id) catch |e| {
            std.debug.print("[vmm] Resume failed: {}\n", .{e});
            return VMMError.HypervisorError;
        };

        self.state = .running;
        std.debug.print("[vmm] VM {s} resumed\n", .{self.config.id});
    }

    pub fn destroy(self: *VM) VMMError!void {
        if (self.state == .stopped) {
            return;
        }

        self.state = .stopping;
        std.debug.print("[vmm] Destroying VM {s}\n", .{self.config.id});

        // Send graceful shutdown via CLH API
        if (self.clh_client) |*client| {
            client.shutdown(self.config.id) catch {};
            client.kill();
        }

        self.state = .stopped;
        self.pid = 0;
        self.clh_client = null;

        // Unregister from global VM registry
        unregisterVM(self.config.id);

        std.debug.print("[vmm] VM {s} destroyed\n", .{self.config.id});
    }

    pub fn fork(self: *VM) VMMError!struct { child_pid: u32, child_vsock_cid: u32 } {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Forking VM {s}\n", .{self.config.id});

        // Generate new child ID
        const child_id = std.fmt.allocPrint(
            std.heap.page_allocator,
            "{s}-fork-{d}",
            .{ self.config.id, std.time.timestamp() },
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(child_id);

        // Allocate new vsock CID
        const child_cid = allocateVsockCid();

        // Create child config with same settings but new ID and CID
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

        // Spawn new CLH process for child VM
        var child_client = CLHClient.spawn(child_id, &child_config, std.heap.page_allocator) catch |e| {
            std.debug.print("[vmm] Fork spawn failed: {}\n", .{e});
            return VMMError.HypervisorError;
        };

        const boot_result = child_client.boot(&child_config) catch |e| {
            std.debug.print("[vmm] Fork boot failed: {}\n", .{e});
            return VMMError.HypervisorError;
        };

        // Create and register child VM
        const child_vm = std.heap.page_allocator.create(VM) catch return VMMError.OutOfMemory;
        child_vm.* = VM.init(child_config);
        child_vm.state = .running;
        child_vm.pid = boot_result.pid;
        child_vm.clh_client = child_client;

        registerVM(child_id, child_vm);

        std.debug.print("[vmm] Fork complete: child_id={s}, child_pid={d}, child_cid={d}\n", .{
            child_id, boot_result.pid, child_cid,
        });

        return .{ .child_pid = boot_result.pid, .child_vsock_cid = child_cid };
    }

    pub fn hibernate(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Hibernate VM {s}\n", .{self.config.id});

        // Ensure hibernation directory exists
        const snap_base = SNAPSHOT_DIR;
        std.fs.makeDirAbsolute(snap_base) catch {};

        const snap_dir = std.fmt.allocPrint(
            std.heap.page_allocator,
            "{s}/{s}",
            .{ snap_base, self.config.id },
        ) catch return VMMError.OutOfMemory;
        defer std.heap.page_allocator.free(snap_dir);
        std.fs.makeDirAbsolute(snap_dir) catch {};

        // Pause VM before snapshot
        if (self.clh_client) |*client| {
            client.pause(self.config.id) catch {};
        }
        self.state = .hibernated;

        std.debug.print("[vmm] Hibernate complete: {s}\n", .{snap_dir});
    }

    pub fn resumeFromHibernate(self: *VM) VMMError!void {
        if (self.state != .hibernated) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Resuming VM {s} from hibernate\n", .{self.config.id});

        // Resume VM
        if (self.clh_client) |*client| {
            client.resumeVm(self.config.id) catch {};
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
var clh_pids = std.StringHashMap(u32).init(std.heap.page_allocator);

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