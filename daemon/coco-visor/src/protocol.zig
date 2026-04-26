// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Binary protocol definitions for visor-node communication.

const std = @import("std");

pub const REQ_BOOT: u32 = 1;
pub const REQ_EXEC: u32 = 2;
pub const REQ_DESTROY: u32 = 3;
pub const REQ_PAUSE: u32 = 4;
pub const REQ_RESUME: u32 = 5;
pub const REQ_GET_STATE: u32 = 6;
pub const REQ_FORK: u32 = 7;
pub const REQ_HIBERNATE: u32 = 8;
pub const REQ_RESUME_HIBERNATED: u32 = 9;
pub const REQ_SNAPSHOT: u32 = 10;
pub const REQ_RESTORE: u32 = 11;

pub const RESP_OK: u32 = 100;
pub const RESP_BOOT: u32 = 101;
pub const RESP_EXEC: u32 = 102;
pub const RESP_DESTROY: u32 = 103;
pub const RESP_GET_STATE: u32 = 106;
pub const RESP_FORK: u32 = 107;
pub const RESP_HIBERNATE: u32 = 108;
pub const RESP_SNAPSHOT: u32 = 110;
pub const RESP_RESTORE: u32 = 111;
pub const RESP_ERROR: u32 = 199;

pub const VM_STATE_CREATED: u32 = 0;
pub const VM_STATE_BOOTING: u32 = 1;
pub const VM_STATE_RUNNING: u32 = 2;
pub const VM_STATE_PAUSED: u32 = 3;
pub const VM_STATE_HIBERNATED: u32 = 4;
pub const VM_STATE_STOPPING: u32 = 5;
pub const VM_STATE_STOPPED: u32 = 6;
pub const VM_STATE_ERROR: u32 = 7;

pub inline fn w32(buf: [*]u8, value: u32) void {
    std.mem.writeInt(u32, @ptrCast(buf), value, .little);
}

pub inline fn r32(buf: []const u8) u32 {
    return std.mem.readInt(u32, @ptrCast(buf.ptr), .little);
}
