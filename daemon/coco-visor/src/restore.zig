// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Checkpoint restoration operations.

const std = @import("std");

pub const RestoreError = error{
    RestoreFailed,
    CheckpointNotFound,
    InvalidCheckpoint,
};

pub fn restoreFromSnapshot(id: []const u8, mem_ptr: [*]u8) !void {
    _ = id;
    _ = mem_ptr;
}
