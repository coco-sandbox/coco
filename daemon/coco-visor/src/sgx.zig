// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Intel SGX (Software Guard Extensions) support.

const std = @import("std");

pub const SGXError = error{
    NotSupported,
    InitFailed,
};

pub fn isSGXSupported() bool {
    return false;
}

pub fn initSGX() SGXError!void {
    return SGXError.NotSupported;
}
