// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Configuration management for cocovisor.

const std = @import("std");

// =============================================================================
// Config
// =============================================================================

pub const Config = struct {
    socket_path: []const u8,
    log_level: []const u8,
    log_file: ?[]const u8,
    max_vms: u32,
    memory_per_vm_mb: u32,
    default_vcpus: u32,
    boot_timeout_ms: u32,
    exec_timeout_ms: u32,

    pub fn default() Config {
        return .{
            .socket_path = "/run/coco/visor.sock",
            .log_level = "info",
            .log_file = null,
            .max_vms = 256,
            .memory_per_vm_mb = 512,
            .default_vcpus = 2,
            .boot_timeout_ms = 30000,
            .exec_timeout_ms = 300000,
        };
    }

    pub fn load(env: *std.process.EnvMap) Config {
        var cfg = default();

        if (env.get("COCOVISOR_SOCKET")) |v| {
            cfg.socket_path = v;
        }
        if (env.get("COCOVISOR_LOG_LEVEL")) |v| {
            cfg.log_level = v;
        }
        if (env.get("COCOVISOR_LOG_FILE")) |v| {
            cfg.log_file = v;
        }
        if (env.get("COCOVISOR_MAX_VMS")) |v| {
            cfg.max_vms = parseInt(u32, v, 10) catch cfg.max_vms;
        }
        if (env.get("COCOVISOR_MEMORY_PER_VM_MB")) |v| {
            cfg.memory_per_vm_mb = parseInt(u32, v, 10) catch cfg.memory_per_vm_mb;
        }
        if (env.get("COCOVISOR_DEFAULT_VCPUS")) |v| {
            cfg.default_vcpus = parseInt(u32, v, 10) catch cfg.default_vcpus;
        }
        if (env.get("COCOVISOR_BOOT_TIMEOUT_MS")) |v| {
            cfg.boot_timeout_ms = parseInt(u32, v, 10) catch cfg.boot_timeout_ms;
        }
        if (env.get("COCOVISOR_EXEC_TIMEOUT_MS")) |v| {
            cfg.exec_timeout_ms = parseInt(u32, v, 10) catch cfg.exec_timeout_ms;
        }

        return cfg;
    }

    fn parseInt(comptime T: type, s: []const u8, base: u8) T {
        return std.fmt.parseInt(T, s, base) catch 0;
    }
};
