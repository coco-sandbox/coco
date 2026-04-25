// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VMM integration with Cloud Hypervisor.
//! Handles MicroVM lifecycle: boot, fork, hibernate, pause, resume.

const std = @import("std");

// =============================================================================
// Constants
// =============================================================================

const SNAPSHOT_DIR = "/var/lib/coco/hibernation";
const CHECKPOINT_DIR = "/var/lib/coco/checkpoints";

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

// =============================================================================
// VM Config
// =============================================================================

pub const VMConfig = struct {
    id: []const u8,
    rootfs: []const u8,
    kernel: []const u8 = "/var/lib/coco/vmlinux",
    initrd: []const u8 = "",
    memory_mb: u32 = 512,
    vcpus: u32 = 2,
    vsock_cid: u32,
    tap_name: []const u8 = "",
};

// =============================================================================
// VM Instance
// =============================================================================

pub const VM = struct {
    config: VMConfig,
    state: VMState,
    pid: u32,
    memory_mb: u32,

    pub fn init(config: VMConfig) VM {
        return .{
            .config = config,
            .state = .created,
            .pid = 0,
            .memory_mb = config.memory_mb,
        };
    }

    pub fn boot(self: *VM) VMMError!struct { pid: u32, vsock_cid: u32 } {
        if (self.state != .created and self.state != .stopped) {
            return VMMError.AlreadyBooted;
        }

        self.state = .booting;
        std.debug.print("[vmm] Booting VM {s} (mem={d}MB, vcpus={d})\n", .{
            self.config.id, self.config.memory_mb, self.config.vcpus,
        });

        // TODO: Use posix.fork() + posix.execvpeZ() to spawn ch-remote
        // For now, just simulate a successful boot with mock PID
        self.pid = 10000 + self.config.vsock_cid * 100;
        self.state = .running;

        std.debug.print("[vmm] VM {s} booted: pid={d}, cid={d}\n", .{
            self.config.id, self.pid, self.config.vsock_cid,
        });

        return .{ .pid = self.pid, .vsock_cid = self.config.vsock_cid };
    }

    pub fn pause(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Pausing VM {s}\n", .{self.config.id});
        // clh-remote pause <vm_id>
        self.state = .paused;
    }

    pub fn resume_(self: *VM) VMMError!void {
        if (self.state != .paused) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Resuming VM {s}\n", .{self.config.id});
        // clh-remote resume <vm_id>
        self.state = .running;
    }

    pub fn destroy(self: *VM) VMMError!void {
        if (self.state == .stopped) {
            return;
        }

        std.debug.print("[vmm] Destroying VM {s}\n", .{self.config.id});
        // clh-remote shutdown <vm_id>
        self.state = .stopped;
        self.pid = 0;
    }

    pub fn fork(self: *VM) VMMError!struct { child_pid: u32, child_vsock_cid: u32 } {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Forking VM {s}\n", .{self.config.id});

        // In production:
        // 1. Cloud Hypervisor snapshot-save (creates memory.img)
        // 2. Clone memory.img via reflink (CoW)
        // 3. Boot new VM from cloned image with unique vsock CID
        const child_pid = self.pid + 1;
        const child_vsock_cid = self.config.vsock_cid + 1;

        std.debug.print("[vmm] Fork complete: child_pid={d}, child_vsock_cid={d}\n", .{
            child_pid, child_vsock_cid,
        });

        return .{ .child_pid = child_pid, .child_vsock_cid = child_vsock_cid };
    }

    pub fn hibernate(self: *VM) VMMError!void {
        if (self.state != .running) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Hibernate VM {s}\n", .{self.config.id});

        // Ensure directory exists
        const snap_path = try std.fmt.allocPrint(
            std.heap.page_allocator,
            "{s}/{s}",
            .{ SNAPSHOT_DIR, self.config.id },
        );
        std.fs.makeDirAbsolute(snap_path) catch return VMMError.IoError;

        // In production:
        // 1. clh-remote pause <vm_id>
        // 2. Copy memory to snapshot file (with compression)
        // 3. Save VM state to vmstate.bin
        // 4. clh-remote stop <vm_id>

        self.state = .hibernated;
        std.debug.print("[vmm] Hibernate complete: {s}\n", .{snap_path});
    }

    pub fn resumeFromHibernate(self: *VM) VMMError!void {
        if (self.state != .hibernated) {
            return VMMError.NotBooted;
        }

        std.debug.print("[vmm] Resuming VM {s} from hibernate\n", .{self.config.id});

        // In production:
        // 1. clh-remote start --restore <snapshot_path>
        // 2. VM resumes from memory.img.zst + vmstate.bin

        self.state = .running;
    }
};

// =============================================================================
// Snapshot Management
// =============================================================================

pub const Snapshot = struct {
    id: []const u8,
    sandbox_id: []const u8,
    path: []const u8,
    memory_mb: u32,
    created_at: i64,

    pub fn delete(self: *Snapshot) VMMError!void {
        std.debug.print("[snapshot] Deleting snapshot {s}\n", .{self.path});
        std.fs.deleteFileAbsolute(self.path) catch {};
        // Also delete memory.img.zst and vmstate.bin
    }
};

// =============================================================================
// Global VM Registry
// =============================================================================

var vms = std.StringHashMap(*VM).init(std.heap.page_allocator);

pub fn getVMs() *std.StringHashMap(*VM) {
    return &vms;
}

pub fn getOrCreateVM(config: VMConfig) VMMError!*VM {
    if (vms.get(config.id)) |existing| {
        return existing;
    }

    const vm = try std.heap.page_allocator.create(VM);
    vm.* = VM.init(config);
    try vms.put(config.id, vm);
    return vm;
}

pub fn removeVM(id: []const u8) void {
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
