const std = @import("std");
const testing = std.testing;

const vmm = @import("../src/vmm.zig");
const checkpoint = @import("../src/checkpoint.zig");
const fork = @import("../src/fork.zig");
const metrics_mod = @import("../src/metrics.zig");
const config = @import("../src/config.zig");
const memory = @import("../src/memory.zig");

test "metrics record boot" {
    var m = metrics_mod.Metrics{};
    m.recordBoot(1000000);
    m.recordBoot(2000000);

    try testing.expectEqual(@as(u64, 2), m.boot_count);
    try testing.expectEqual(@as(u64, 3000000), m.total_boot_time_ns);
    try testing.expectEqual(@as(u64, 1500000), m.avgBootTimeNs());
}

test "metrics record fork" {
    var m = metrics_mod.Metrics{};
    m.recordFork(500000);
    m.recordFork(1500000);

    try testing.expectEqual(@as(u64, 2), m.fork_count);
    try testing.expectEqual(@as(u64, 1000000), m.avgForkTimeNs());
}

test "metrics sandbox counters" {
    var m = metrics_mod.Metrics{};
    m.incActiveSandboxes();
    m.incActiveSandboxes();
    m.incPausedSandboxes();
    m.incHibernatedSandboxes();

    try testing.expectEqual(@as(u64, 2), m.active_sandboxes);
    try testing.expectEqual(@as(u64, 1), m.paused_sandboxes);
    try testing.expectEqual(@as(u64, 1), m.hibernated_sandboxes);

    m.decActiveSandboxes();
    try testing.expectEqual(@as(u64, 1), m.active_sandboxes);
}

test "config defaults" {
    const cfg = config.loadConfig();

    try testing.expectEqualStrings("/run/coco/visor.sock", cfg.socket_path);
    try testing.expectEqual(@as(u32, 512), cfg.memory_mb);
    try testing.expectEqual(@as(u32, 2), cfg.vcpus);
    try testing.expectEqual(@as(u16, 9090), cfg.metrics_port);
    try testing.expectEqual(@as(u16, 4748), cfg.health_port);
}

test "checkpoint metadata" {
    const meta = checkpoint.CheckpointMetadata{
        .id = "test-vm",
        .memory_size = 512 * 1024 * 1024,
        .compressed_size = 256 * 1024 * 1024,
        .timestamp = 1234567890,
        .memory_mb = 512,
        .vcpus = 2,
        .kernel = "/var/lib/coco/vmlinux",
        .rootfs = "/var/lib/coco/templates/ubuntu",
    };

    try testing.expectEqualStrings("test-vm", meta.id);
    try testing.expectEqual(@as(u64, 512 * 1024 * 1024), meta.memory_size);
}

test "fork manager init" {
    const allocator = std.heap.page_allocator;
    const fm = fork.ForkManager.init(allocator, "/tmp/coco-test");

    try testing.expect(fm.base_dir.len > 0);
}

test "memory guest memory slice" {
    const size: u64 = 1024 * 1024;
    const ptr = std.heap.page_allocator.alloc(u8, size) catch return;
    defer std.heap.page_allocator.free(ptr);

    const mem = memory.GuestMemory{
        .ptr = ptr.ptr,
        .size = size,
    };

    const slice = mem.sliceAt(0x1000, 512);
    try testing.expectEqual(@as(usize, 512), slice.len);

    const host_ptr = mem.hostAddr(0x1000);
    try testing.expectEqual(ptr.ptr + 0x1000, host_ptr);
}

test "vm state enum" {
    try testing.expectEqual(@as(u32, 0), @intFromEnum(vmm.VMState.created));
    try testing.expectEqual(@as(u32, 1), @intFromEnum(vmm.VMState.booting));
    try testing.expectEqual(@as(u32, 2), @intFromEnum(vmm.VMState.running));
    try testing.expectEqual(@as(u32, 3), @intFromEnum(vmm.VMState.paused));
    try testing.expectEqual(@as(u32, 4), @intFromEnum(vmm.VMState.hibernated));
}
