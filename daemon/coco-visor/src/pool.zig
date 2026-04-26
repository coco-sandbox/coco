// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! VM Pool implementation for sub-30ms cold starts.
//! Spec: Pre-created VMs maintained in pool for immediate assignment.

const std = @import("std");
const vmm = @import("vmm.zig");

const PooledVm = struct {
    vm: *vmm.VM,
    ready: bool,
    last_used: i64,
};

pub const VmPool = struct {
    allocator: std.mem.Allocator,
    pool: std.ArrayList(PooledVm),
    config: vmm.VMConfig,
    max_size: u32,
    precreate: bool,
    mutex: std.Thread.Mutex,

    pub fn init(allocator: std.mem.Allocator, config: vmm.VMConfig, max_size: u32, precreate: bool) !VmPool {
        var pool = VmPool{
            .allocator = allocator,
            .pool = std.ArrayList(PooledVm).init(allocator),
            .config = config,
            .max_size = max_size,
            .precreate = precreate,
            .mutex = .{},
        };

        if (precreate) {
            try pool.precreateVMs(max_size);
        }

        return pool;
    }

    pub fn deinit(self: *VmPool) void {
        for (self.pool.items) |item| {
            item.vm.destroy() catch {};
            self.allocator.destroy(item.vm);
        }
        self.pool.deinit();
    }

    fn precreateVMs(self: *VmPool, count: u32) !void {
        var i: u32 = 0;
        while (i < count) : (i += 1) {
            const vm = self.allocator.create(vmm.VM) catch continue;

            var vm_config = self.config;
            vm_config.vsock_cid = vmm.allocateVsockCid();
            vm.* = vmm.VM.init(vm_config);

            const result = vm.boot() catch {
                self.allocator.destroy(vm);
                continue;
            };

            try self.pool.append(.{
                .vm = vm,
                .ready = true,
                .last_used = sc.timestamp(),
            });

            _ = result;
        }
    }

    pub fn acquire(self: *VmPool) ?*vmm.VM {
        self.mutex.lock();
        defer self.mutex.unlock();

        for (self.pool.items, 0..) |item, idx| {
            if (item.ready) {
                self.pool.items[idx].ready = false;
                self.pool.items[idx].last_used = sc.timestamp();
                return item.vm;
            }
        }

        if (self.pool.items.len < self.max_size) {
            const vm = self.allocator.create(vmm.VM) catch return null;

            var vm_config = self.config;
            vm_config.vsock_cid = vmm.allocateVsockCid();
            vm.* = vmm.VM.init(vm_config);

            vm.boot() catch {
                self.allocator.destroy(vm);
                return null;
            };

            self.pool.append(.{
                .vm = vm,
                .ready = false,
                .last_used = sc.timestamp(),
            }) catch return vm;

            return vm;
        }

        return null;
    }

    pub fn release(self: *VmPool, vm: *vmm.VM) void {
        self.mutex.lock();
        defer self.mutex.unlock();

        for (self.pool.items, 0..) |item, idx| {
            if (item.vm == vm) {
                self.pool.items[idx].ready = true;
                self.pool.items[idx].last_used = sc.timestamp();
                return;
            }
        }
    }

    pub fn availableCount(self: *VmPool) u32 {
        self.mutex.lock();
        defer self.mutex.unlock();

        var count: u32 = 0;
        for (self.pool.items) |item| {
            if (item.ready) count += 1;
        }
        return count;
    }

    pub fn totalCount(self: *VmPool) u32 {
        self.mutex.lock();
        defer self.mutex.unlock();
        return @intCast(self.pool.items.len);
    }

    pub fn cleanupStale(self: *VmPool, max_age_seconds: i64) void {
        self.mutex.lock();
        defer self.mutex.unlock();

        const now = sc.timestamp();
        var to_remove: usize = 0;

        for (self.pool.items) |item| {
            if (item.ready and now - item.last_used > max_age_seconds) {
                to_remove += 1;
            }
        }

        if (to_remove > 0) {
            var new_pool = std.ArrayList(PooledVm).init(self.allocator);
            for (self.pool.items) |item| {
                if (item.ready and now - item.last_used > max_age_seconds) {
                    item.vm.destroy() catch {};
                    self.allocator.destroy(item.vm);
                } else {
                    new_pool.append(item) catch {};
                }
            }
            self.pool.deinit();
            self.pool = new_pool;
        }
    }

    pub fn refill(self: *VmPool) void {
        self.mutex.lock();
        defer self.mutex.unlock();

        const current: u32 = @intCast(self.pool.items.len);
        if (current < self.max_size) {
            const needed = self.max_size - current;
            const to_create = @min(needed, 4);

            var i: u32 = 0;
            while (i < to_create) : (i += 1) {
                const vm = self.allocator.create(vmm.VM) catch continue;

                var vm_config = self.config;
                vm_config.vsock_cid = vmm.allocateVsockCid();
                vm.* = vmm.VM.init(vm_config);

                vm.boot() catch {
                    self.allocator.destroy(vm);
                    continue;
                };

                self.pool.append(.{
                    .vm = vm,
                    .ready = true,
                    .last_used = sc.timestamp(),
                }) catch {};
            }
        }
    }
};

var global_pool: ?*VmPool = null;
var pool_mutex: std.Thread.Mutex = .{};

pub fn initPool(allocator: std.mem.Allocator, config: vmm.VMConfig, max_size: u32, precreate: bool) !void {
    pool_mutex.lock();
    defer pool_mutex.unlock();

    if (global_pool) |p| {
        p.deinit();
        allocator.destroy(p);
    }

    global_pool = try allocator.create(VmPool);
    global_pool.?.* = try VmPool.init(allocator, config, max_size, precreate);
}

pub fn getPool() ?*VmPool {
    pool_mutex.lock();
    defer pool_mutex.unlock();
    return global_pool;
}

pub fn destroyPool() void {
    pool_mutex.lock();
    defer pool_mutex.unlock();

    if (global_pool) |p| {
        p.deinit();
        global_pool = null;
    }
}

pub fn acquireVm() ?*vmm.VM {
    if (getPool()) |pool| {
        return pool.acquire();
    }
    return null;
}

pub fn releaseVm(vm: *vmm.VM) void {
    if (getPool()) |pool| {
        pool.release(vm);
    }
}
