// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Configuration for coco-visor.
//! Spec: Configurable via file, environment variables, and defaults.

const std = @import("std");

pub const LogLevel = enum { debug, info, warn, err };

pub const Config = struct {
    socket_path: []const u8 = "/run/coco/visor.sock",
    memory_mb: u32 = 512,
    vcpus: u32 = 2,
    kernel_path: []const u8 = "/var/lib/coco/vmlinux",
    initrd_path: []const u8 = "",
    vsock_port: u32 = 4747,
    template_dir: []const u8 = "/var/lib/coco/templates",
    checkpoint_dir: []const u8 = "/var/lib/coco/checkpoints",
    hibernation_dir: []const u8 = "/var/lib/coco/hibernation",
    metrics_enabled: bool = true,
    metrics_port: u16 = 9090,
    health_enabled: bool = true,
    health_port: u16 = 4748,
    log_level: []const u8 = "info",
    pool_size: u32 = 4,
    pool_precreate: bool = true,
    compression_level: i32 = 3,
    max_concurrent_forks: u32 = 8,
};

pub fn loadConfig() Config {
    var config = Config{};

    const config_paths = [_][]const u8{
        "/etc/coco/visor.yaml",
        "/etc/coco/visor.yml",
        "/etc/coco/config.yaml",
        "/etc/coco/config.yml",
    };

    for (config_paths) |path| {
        if (loadConfigFile(path, &config)) {
            break;
        }
    }

    applyEnvOverrides(&config);

    return config;
}

fn loadConfigFile(path: []const u8, config: *Config) bool {
    const file = std.fs.openFileAbsolute(path, .{}) catch return false;
    defer file.close();

    const content = file.readToEndAlloc(std.heap.page_allocator, 8192) catch return false;
    defer std.heap.page_allocator.free(content);

    parseYaml(content, config) catch return false;
    return true;
}

fn parseYaml(content: []const u8, config: *Config) !void {
    var lines = std.mem.splitScalar(u8, content, '\n');

    while (lines.next()) |line| {
        const trimmed = std.mem.trim(u8, line, " \t");
        if (trimmed.len == 0 or trimmed[0] == '#') continue;

        if (std.mem.startsWith(u8, trimmed, "socket_path:")) {
            const value = std.mem.trim(u8, trimmed[12..], " \"");
            config.socket_path = try std.heap.page_allocator.dupe(u8, value);
        } else if (std.mem.startsWith(u8, trimmed, "memory_mb:")) {
            const value = std.mem.trim(u8, trimmed[10..], " \"");
            config.memory_mb = std.fmt.parseInt(u32, value, 10) catch config.memory_mb;
        } else if (std.mem.startsWith(u8, trimmed, "vcpus:")) {
            const value = std.mem.trim(u8, trimmed[6..], " \"");
            config.vcpus = std.fmt.parseInt(u32, value, 10) catch config.vcpus;
        } else if (std.mem.startsWith(u8, trimmed, "kernel_path:")) {
            const value = std.mem.trim(u8, trimmed[12..], " \"");
            config.kernel_path = try std.heap.page_allocator.dupe(u8, value);
        } else if (std.mem.startsWith(u8, trimmed, "template_dir:")) {
            const value = std.mem.trim(u8, trimmed[12..], " \"");
            config.template_dir = try std.heap.page_allocator.dupe(u8, value);
        } else if (std.mem.startsWith(u8, trimmed, "checkpoint_dir:")) {
            const value = std.mem.trim(u8, trimmed[14..], " \"");
            config.checkpoint_dir = try std.heap.page_allocator.dupe(u8, value);
        } else if (std.mem.startsWith(u8, trimmed, "hibernation_dir:")) {
            const value = std.mem.trim(u8, trimmed[16..], " \"");
            config.hibernation_dir = try std.heap.page_allocator.dupe(u8, value);
        } else if (std.mem.startsWith(u8, trimmed, "vsock_port:")) {
            const value = std.mem.trim(u8, trimmed[10..], " \"");
            config.vsock_port = std.fmt.parseInt(u32, value, 10) catch config.vsock_port;
        } else if (std.mem.startsWith(u8, trimmed, "metrics_port:")) {
            const value = std.mem.trim(u8, trimmed[12..], " \"");
            config.metrics_port = std.fmt.parseInt(u16, value, 10) catch config.metrics_port;
        } else if (std.mem.startsWith(u8, trimmed, "health_port:")) {
            const value = std.mem.trim(u8, trimmed[11..], " \"");
            config.health_port = std.fmt.parseInt(u16, value, 10) catch config.health_port;
        } else if (std.mem.startsWith(u8, trimmed, "log_level:")) {
            const value = std.mem.trim(u8, trimmed[9..], " \"");
            config.log_level = try std.heap.page_allocator.dupe(u8, value);
        } else if (std.mem.startsWith(u8, trimmed, "pool_size:")) {
            const value = std.mem.trim(u8, trimmed[9..], " \"");
            config.pool_size = std.fmt.parseInt(u32, value, 10) catch config.pool_size;
        } else if (std.mem.startsWith(u8, trimmed, "pool_precreate:")) {
            const value = std.mem.trim(u8, trimmed[14..], " \"");
            config.pool_precreate = std.mem.eql(u8, value, "true");
        }
    }
}

fn applyEnvOverrides(config: *Config) void {
    _ = config;
}

pub fn getLogLevel(config: *const Config) LogLevel {
    if (std.mem.eql(u8, config.log_level, "debug")) return .debug;
    if (std.mem.eql(u8, config.log_level, "warn")) return .warn;
    if (std.mem.eql(u8, config.log_level, "err")) return .err;
    return .info;
}
