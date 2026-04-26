// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Intel TDX (Trust Domain Extensions) support.

const std = @import("std");

pub const TDXError = error{
    NotSupported,
    InitFailed,
};

pub fn isTDXSupported() bool {
    return false;
}

pub fn initTDX() TDXError!void {
    return TDXError.NotSupported;
}
