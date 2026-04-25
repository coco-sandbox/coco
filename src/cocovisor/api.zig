// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! API definitions for cocovisor.
//! Provides request/response types for the Unix socket protocol.

const std = @import("std");
const vmm = @import("vmm");

// =============================================================================
// Protocol Constants
// =============================================================================

pub const REQ_BOOT: u32 = 1;
pub const REQ_EXEC: u32 = 2;
pub const REQ_DESTROY: u32 = 3;
pub const REQ_PAUSE: u32 = 4;
pub const REQ_RESUME: u32 = 5;
pub const REQ_GET_STATE: u32 = 6;
pub const REQ_FORK: u32 = 7;
pub const REQ_HIBERNATE: u32 = 8;
pub const REQ_RESUME_HIBERNATED: u32 = 9;

pub const RESP_OK: u32 = 100;
pub const RESP_BOOT: u32 = 101;
pub const RESP_EXEC: u32 = 102;
pub const RESP_DESTROY: u32 = 103;
pub const RESP_PAUSE: u32 = 104;
pub const RESP_RESUME: u32 = 105;
pub const RESP_GET_STATE: u32 = 106;
pub const RESP_FORK: u32 = 107;
pub const RESP_HIBERNATE: u32 = 108;
pub const RESP_ERROR: u32 = 199;

// =============================================================================
// Request Structures
// =============================================================================

pub const BootRequest = extern struct {
    rootfs_path_len: u32,
    memory_mb: u32,
    vcpu_count: u32,
    kernel_path_len: u32,
    initrd_path_len: u32,
    sandbox_id_len: u32,
    vsock_port: u32,
    padding: u32,
};

pub const ExecRequest = extern struct {
    cmd_len: u32,
    args_len: u32,
    env_len: u32,
    working_dir_len: u32,
};

// =============================================================================
// Response Structures
// =============================================================================

pub const BootResponse = struct {
    vsock_cid: u32,
    pid: u32,
    state: u32,
};

pub const ForkResponse = struct {
    child_vsock_cid: u32,
    child_pid: u32,
    duration_ms: u32,
};

pub const GetStateResponse = struct {
    state: u32,
    pid: u32,
    vsock_cid: u32,
};

pub const ExecStreamChunk = struct {
    stream_type: u32, // 1=stdout, 2=stderr, 3=exit, 4=signal
    data_len: u32,
    exit_code: u32,
};

// =============================================================================
// Protocol Helpers
// =============================================================================

pub fn writeResponseHeader(buf: []u8, kind: u32, size: u32) void {
    std.mem.writeInt(u32, buf[0..4], kind, .little);
    std.mem.writeInt(u32, buf[4..8], size, .little);
}

pub fn readResponseKind(buf: []const u8) u32 {
    return std.mem.readInt(u32, buf[0..4], .little);
}

pub fn readResponseSize(buf: []const u8) u32 {
    return std.mem.readInt(u32, buf[4..8], .little);
}
