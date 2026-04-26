// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Intel SGX (Software Guard Extensions) support.
//! Spec: SGX provides even stronger isolation than TDX, with protected memory regions.

const std = @import("std");
const linux = std.os.linux;

pub const SGXError = error{
    NotSupported,
    InitFailed,
    EnclaveCreateFailed,
    PageMapFailed,
};

pub const SGXInfo = struct {
    supported: bool,
    enclave_supported: bool,
    epcm_supported: bool,
};

fn cpuid(eax: u32, ecx: u32) struct { eax: u32, ebx: u32, ecx: u32, edx: u32 } {
    var eax_out: u32 = 0;
    var ebx_out: u32 = 0;
    var ecx_out: u32 = 0;
    var edx_out: u32 = 0;

    asm volatile ("cpuid"
        : [eax] "=a" (eax_out),
          [ebx] "=b" (ebx_out),
          [ecx] "=c" (ecx_out),
          [edx] "=d" (edx_out),
        : [eax_in] "a" (eax),
          [ecx_in] "c" (ecx),
    );

    return .{ .eax = eax_out, .ebx = ebx_out, .ecx = ecx_out, .edx = edx_out };
}

pub fn isSGXSupported() bool {
    const result = cpuid(7, 0);
    return (result.ebx & (1 << 2)) != 0;
}

pub fn isKVMSGXSupported(kvm_fd: i32) bool {
    if (!isSGXSupported()) return false;

    const KVM_CAP_SGX: u32 = 0;
    const rc = linux.ioctl(kvm_fd, 0x4000ae03, KVM_CAP_SGX);
    return rc >= 0;
}

pub fn initSGX() SGXError!void {
    if (!isSGXSupported()) {
        return SGXError.NotSupported;
    }
}

pub fn getSGXInfo() SGXInfo {
    const result = cpuid(7, 0);
    return .{
        .supported = isSGXSupported(),
        .enclave_supported = (result.ebx & (1 << 2)) != 0,
        .epcm_supported = (result.ebx & (1 << 1)) != 0,
    };
}

const KVM_SGX_ATTR_PROENCLAVE: u64 = 0x1;
const KVM_SGX_ATTR_PEERENCLAVE: u64 = 0x2;

pub fn createSGXEnclave(vm_fd: i32) SGXError!i32 {
    const KVM_CREATE_ENCLAVE: u32 = 0x4018ae80;

    var attr: u64 = KVM_SGX_ATTR_PROENCLAVE;
    const rc = linux.ioctl(vm_fd, KVM_CREATE_ENCLAVE, @intFromPtr(&attr));

    if (rc < 0) {
        return SGXError.EnclaveCreateFailed;
    }

    return @intCast(rc);
}

pub fn mapEnclavePages(
    _: i32,
    enclave_fd: i32,
    offset: u64,
    src: u64,
    length: u64,
) SGXError!void {
    const KVM_ENCLAVE_ADD_PAGES: u32 = 0x4018ae81;

    const EnclavePageParams = extern struct {
        offset: u64,
        src: u64,
        length: u64,
    };

    const params = EnclavePageParams{
        .offset = offset,
        .src = src,
        .length = length,
    };

    const rc = linux.ioctl(enclave_fd, KVM_ENCLAVE_ADD_PAGES, @intFromPtr(&params));
    if (rc < 0) {
        return SGXError.PageMapFailed;
    }
}

pub fn initEnclave(enclave_fd: i32, entry: u64, tls: u64) SGXError!void {
    const KVM_ENCLAVE_INIT: u32 = 0x4018ae82;

    const EnclaveInitParams = extern struct {
        entry: u64,
        tls: u64,
    };

    const params = EnclaveInitParams{
        .entry = entry,
        .tls = tls,
    };

    const rc = linux.ioctl(enclave_fd, KVM_ENCLAVE_INIT, @intFromPtr(&params));
    if (rc < 0) {
        return SGXError.InitFailed;
    }
}
