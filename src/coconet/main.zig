// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Coconet — Network daemon with eBPF NAT and AF_XDP fast path.
//!
//! Architecture:
//!   - from_sandbox.bpf.c (C) - Egress SNAT, attached via TC
//!   - from_world.bpf.c (C) - Ingress DNAT, attached via TC
//!   - AF_XDP fast path for Intel E810 (bypasses kernel stack)
//!
//! In production, eBPF programs are compiled with:
//!   clang -O2 -target bpf -c c/from_sandbox.bpf.c
//!   clang -O2 -target bpf -c c/from_world.bpf.c
//!
//! Maps are created and managed via bpf(2) syscalls.

const std = @import("std");

// =============================================================================
// Constants
// =============================================================================

const AF_XDP_FLAGS = 2; // XDP_USE_NEED_WAKEUP
const XDP_ACTION_DROP = 0;
const XDP_ACTION_PASS = 2;
const XDP_ACTION_TX = 3;

const NS_PER_SEC: u64 = 1_000_000_000;

const BPF_MAP_TYPE_HASH = 1;
const BPF_MAP_TYPE_PERCPU_ARRAY = 2;
const BPF_MAP_TYPE_XSKMAP = 4;

const EGRESS_SESSIONS_SIZE = 64 * 1024;
const INGRESS_SESSIONS_SIZE = 64 * 1024;

// =============================================================================
// Tuple for session lookup (5-tuple NAT)
// =============================================================================

const Tuple5 = extern struct {
    proto: u8,
    _: [3]u8 = undefined,
    saddr: [4]u8,
    daddr: [4]u8,
    sport: u16,
    dport: u16,
};

const SNATEntry = struct {
    translated_addr: [4]u8,
    translated_port: u16,
    orig_port: u16,
    timestamp: u64,
};

// =============================================================================
// Network Configuration
// =============================================================================

const NetworkConfig = struct {
    host_iface: []const u8,
    sandbox_iface_prefix: []const u8,
    host_ip: [4]u8,
    sandbox_ip_base: [4]u8,
    nat_pool_start: u16,
    nat_pool_end: u16,
};

// =============================================================================
// Policy Engine (Bloom + LPM)
// =============================================================================

const PolicyRule = struct {
    cidr: [2]u64, // 128-bit CIDR prefix
    prefix_len: u32,
    action: u32, // 0=allow, 1=deny
    priority: u32,
};

const PolicyEngine = struct {
    rules: []PolicyRule,
    bloom_filter: [16]u64, // 1024-bit bloom
};

fn bloomInsert(engine: *PolicyEngine, cidr: [2]u64) void {
    _ = engine;
    _ = cidr;
    // Hash 3 times and set bits
    std.debug.print("[coconet] Bloom insert: TODO\n", .{});
}

fn bloomCheck(engine: *PolicyEngine, cidr: [2]u64) bool {
    _ = engine;
    _ = cidr;
    return true;
}

fn lpmMatch(engine: *PolicyEngine, cidr: [2]u64) ?PolicyRule {
    for (engine.rules) |rule| {
        if (cidrMatches(cidr, rule.cidr, rule.prefix_len)) {
            return rule;
        }
    }
    return null;
}

fn cidrMatches(input: [2]u64, prefix: [2]u64, prefix_len: u32) bool {
    if (prefix_len == 0) return true;
    if (prefix_len <= 64) {
        const mask: u64 = (~@as(u64, 0)) << (64 - prefix_len);
        return (input[0] & mask) == (prefix[0] & mask);
    } else {
        if ((input[0] & ~@as(u64, 0)) != (prefix[0] & ~@as(u64, 0))) return false;
        const bits2 = prefix_len - 64;
        const mask2: u64 = (~@as(u64, 0)) << (64 - bits2);
        return (input[1] & mask2) == (prefix[1] & mask2);
    }
}

// =============================================================================
// eBPF Program Management (via bpf syscall)
// =============================================================================

const BpfCmd = enum(u32) {
    create_map = 0,
    lookupElem = 1,
    updateElem = 2,
    deleteElem = 3,
    getNextKey = 4,
    loadProgram = 5,
    attach = 6,
};

const BpfMapCreate = struct {
    map_type: u32,
    key_size: u32,
    value_size: u32,
    max_entries: u32,
    flags: u32,
};

// =============================================================================
// Network Interface Management
// =============================================================================

/// setupSandboxInterface creates a veth pair for a sandbox
fn setupSandboxInterface(sandboxID: []const u8, ip: [4]u8) !void {
    _ = sandboxID;
    _ = ip;
    std.debug.print("[coconet] Creating interface for sandbox\n", .{});
    // In real implementation:
    // ip link add veth_sb_xxx type veth peer name eth_sb_xxx
    // ip addr add 10.0.0.x/24 dev veth_sb_xxx
    // ip link set eth_sb_xxx netns <sandbox_ns>
}

/// teardownSandboxInterface removes veth pair
fn teardownSandboxInterface(sandboxID: []const u8) !void {
    _ = sandboxID;
    std.debug.print("[coconet] Tearing down interface for sandbox\n", .{});
}

// =============================================================================
// NAT Session Management
// =============================================================================

/// createEgressSession creates a SNAT session for outbound traffic
fn createEgressSession(srcIP: [4]u8, srcPort: u16, dstIP: [4]u8, dstPort: u16, proto: u8) !void {
    _ = srcIP;
    _ = srcPort;
    _ = dstIP;
    _ = dstPort;
    _ = proto;
    std.debug.print("[coconet] Creating egress SNAT session\n", .{});
    // bpf(BPF_MAP_UPDATE_ELEM, &egress_sessions, &key, &entry, BPF_ANY)
}

/// createIngressSession creates a DNAT session for inbound traffic
fn createIngressSession(dstIP: [4]u8, dstPort: u16, sandboxIP: [4]u8, sandboxPort: u16, proto: u8) !void {
    _ = dstIP;
    _ = dstPort;
    _ = sandboxIP;
    _ = sandboxPort;
    _ = proto;
    std.debug.print("[coconet] Creating ingress DNAT session\n", .{});
    // bpf(BPF_MAP_UPDATE_ELEM, &ingress_sessions, &key, &entry, BPF_ANY)
}

/// cleanupSession removes a NAT session
fn cleanupSession(key: Tuple5) !void {
    _ = key;
    std.debug.print("[coconet] Cleaning up NAT session\n", .{});
    // bpf(BPF_MAP_DELETE_ELEM, &sessions, &key, 0)
}

// =============================================================================
// AF_XDP Fast Path
// =============================================================================

/// setupXDPFastPath configures AF_XDP for Intel E810
fn setupXDPFastPath(iface: []const u8, queue: u32) !void {
    _ = iface;
    _ = queue;
    std.debug.print("[coconet] Setting up AF_XDP fast path\n", .{});
    // In real implementation:
    // struct xdp_socket *xsk = xsk_socket__create(&xdp_desc, queue, ifindex, &xsk_config);
    // bpf(BPF_MAP_UPDATE_ELEM, &xsks, &queue, &xsk, BPF_ANY);
}

/// attachTCPrograms attaches eBPF TC classifiers
fn attachTCPrograms(iface: []const u8) !void {
    _ = iface;
    std.debug.print("[coconet] Attaching TC eBPF programs\n", .{});
    // tc qdisc add dev <iface> clsact
    // tc filter add dev <iface> egress bpf da obj c/from_sandbox.bpf.o sec tc
    // tc filter add dev <iface> ingress bpf da obj c/from_world.bpf.o sec tc
}

// =============================================================================
// Main
// =============================================================================

pub fn main() !void {
    std.debug.print("[coconet] Starting network daemon\n", .{});
    std.debug.print("[coconet] AF_XDP + eBPF NAT engine\n", .{});
    std.debug.print("[coconet] Policies: Bloom filter + LPM\n", .{});

    std.debug.print("[coconet] Daemon ready (placeholder mode)\n", .{});

    // Block forever
    while (true) {
        std.time.sleep(NS_PER_SEC);
    }
}
