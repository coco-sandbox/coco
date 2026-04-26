// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Configuration for coco-visor.
//! Spec: Configurable via file, environment variables, and defaults.

const std = @import("std");
const sc = @import("syscall.zig");
const linux = std.os.linux;

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
        if (loadConfigFile(path, &config)) break;
    }

    applyEnvOverrides(&config);
    return config;
}

fn loadConfigFile(path: []const u8, config: *Config) bool {
    const fd = sc.open(path, .{ .ACCMODE = .RDONLY }, 0) catch return false;
    defer sc.close(fd);

    var buf: [8192]u8 = undefined;
    const n = sc.read(fd, &buf) catch return false;
    if (n == 0) return false;
    parseYaml(buf[0..n], config) catch return false;
    return true;
}

fn matchKey(line: []const u8, key: []const u8) ?[]const u8 {
    if (!std.mem.startsWith(u8, line, key)) return null;
    if (line.len <= key.len) return "";
    if (line[key.len] != ':') return null;
    return std.mem.trim(u8, line[key.len + 1 ..], " \t\"'");
}

fn parseYaml(content: []const u8, config: *Config) !void {
    var lines = std.mem.splitScalar(u8, content, '\n');
    const allocator = std.heap.page_allocator;

    while (lines.next()) |line| {
        const trimmed = std.mem.trim(u8, line, " \t\r");
        if (trimmed.len == 0 or trimmed[0] == '#') continue;

        if (matchKey(trimmed, "socket_path")) |v| {
            config.socket_path = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "memory_mb")) |v| {
            config.memory_mb = std.fmt.parseInt(u32, v, 10) catch config.memory_mb;
        } else if (matchKey(trimmed, "vcpus")) |v| {
            config.vcpus = std.fmt.parseInt(u32, v, 10) catch config.vcpus;
        } else if (matchKey(trimmed, "kernel_path")) |v| {
            config.kernel_path = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "initrd_path")) |v| {
            config.initrd_path = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "template_dir")) |v| {
            config.template_dir = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "checkpoint_dir")) |v| {
            config.checkpoint_dir = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "hibernation_dir")) |v| {
            config.hibernation_dir = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "vsock_port")) |v| {
            config.vsock_port = std.fmt.parseInt(u32, v, 10) catch config.vsock_port;
        } else if (matchKey(trimmed, "metrics_enabled")) |v| {
            config.metrics_enabled = std.mem.eql(u8, v, "true");
        } else if (matchKey(trimmed, "metrics_port")) |v| {
            config.metrics_port = std.fmt.parseInt(u16, v, 10) catch config.metrics_port;
        } else if (matchKey(trimmed, "health_enabled")) |v| {
            config.health_enabled = std.mem.eql(u8, v, "true");
        } else if (matchKey(trimmed, "health_port")) |v| {
            config.health_port = std.fmt.parseInt(u16, v, 10) catch config.health_port;
        } else if (matchKey(trimmed, "log_level")) |v| {
            config.log_level = try allocator.dupe(u8, v);
        } else if (matchKey(trimmed, "pool_size")) |v| {
            config.pool_size = std.fmt.parseInt(u32, v, 10) catch config.pool_size;
        } else if (matchKey(trimmed, "pool_precreate")) |v| {
            config.pool_precreate = std.mem.eql(u8, v, "true");
        } else if (matchKey(trimmed, "compression_level")) |v| {
            config.compression_level = std.fmt.parseInt(i32, v, 10) catch config.compression_level;
        } else if (matchKey(trimmed, "max_concurrent_forks")) |v| {
            config.max_concurrent_forks = std.fmt.parseInt(u32, v, 10) catch config.max_concurrent_forks;
        }
    }
}

fn getEnv(key: [*:0]const u8) ?[]const u8 {
    const c_str = std.c.getenv(key);
    if (c_str == null) return null;
    return std.mem.span(c_str.?);
}

fn applyEnvOverrides(config: *Config) void {
    const allocator = std.heap.page_allocator;

    if (getEnv("COCO_VISOR_SOCKET_PATH")) |v| {
        config.socket_path = allocator.dupe(u8, v) catch config.socket_path;
    }
    if (getEnv("COCO_VISOR_MEMORY_MB")) |v| {
        config.memory_mb = std.fmt.parseInt(u32, v, 10) catch config.memory_mb;
    }
    if (getEnv("COCO_VISOR_VCPUS")) |v| {
        config.vcpus = std.fmt.parseInt(u32, v, 10) catch config.vcpus;
    }
    if (getEnv("COCO_VISOR_KERNEL_PATH")) |v| {
        config.kernel_path = allocator.dupe(u8, v) catch config.kernel_path;
    }
    if (getEnv("COCO_VISOR_INITRD_PATH")) |v| {
        config.initrd_path = allocator.dupe(u8, v) catch config.initrd_path;
    }
    if (getEnv("COCO_VISOR_VSOCK_PORT")) |v| {
        config.vsock_port = std.fmt.parseInt(u32, v, 10) catch config.vsock_port;
    }
    if (getEnv("COCO_VISOR_METRICS_PORT")) |v| {
        config.metrics_port = std.fmt.parseInt(u16, v, 10) catch config.metrics_port;
    }
    if (getEnv("COCO_VISOR_HEALTH_PORT")) |v| {
        config.health_port = std.fmt.parseInt(u16, v, 10) catch config.health_port;
    }
    if (getEnv("COCO_VISOR_LOG_LEVEL")) |v| {
        config.log_level = allocator.dupe(u8, v) catch config.log_level;
    }
    if (getEnv("COCO_VISOR_POOL_SIZE")) |v| {
        config.pool_size = std.fmt.parseInt(u32, v, 10) catch config.pool_size;
    }
    if (getEnv("COCO_VISOR_POOL_PRECREATE")) |v| {
        config.pool_precreate = std.mem.eql(u8, v, "true");
    }
    if (getEnv("COCO_VISOR_METRICS_ENABLED")) |v| {
        config.metrics_enabled = std.mem.eql(u8, v, "true");
    }
    if (getEnv("COCO_VISOR_HEALTH_ENABLED")) |v| {
        config.health_enabled = std.mem.eql(u8, v, "true");
    }
    if (getEnv("COCO_VISOR_COMPRESSION_LEVEL")) |v| {
        config.compression_level = std.fmt.parseInt(i32, v, 10) catch config.compression_level;
    }
    if (getEnv("COCO_VISOR_MAX_CONCURRENT_FORKS")) |v| {
        config.max_concurrent_forks = std.fmt.parseInt(u32, v, 10) catch config.max_concurrent_forks;
    }
}

pub fn getLogLevel(config: *const Config) LogLevel {
    if (std.mem.eql(u8, config.log_level, "debug")) return .debug;
    if (std.mem.eql(u8, config.log_level, "warn")) return .warn;
    if (std.mem.eql(u8, config.log_level, "err")) return .err;
    return .info;
}
