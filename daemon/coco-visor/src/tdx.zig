// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Intel TDX (Trust Domain Extensions) support.
//! Spec: Hardware enclaves provide additional protection for sensitive workloads.

const std = @import("std");
const linux = std.os.linux;

pub const TDXError = error{
    NotSupported,
    InitFailed,
    AttestationFailed,
};

pub const TDXInfo = struct {
    trusted: bool,
    attestation_supported: bool,
};

const TDX_GET_QUOTE: u32 = 0x1001;
const TDX_SYS_CPUID_LEAF: u32 = 0x21;

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

pub fn isTDXSupported() bool {
    const result = cpuid(TDX_SYS_CPUID_LEAF, 0);
    return result.eax == 0x54445058;
}

pub fn isKVMTDXSupported(kvm_fd: i32) bool {
    if (!isTDXSupported()) return false;

    const TDX_CAPABILITIES: u32 = 0x1001;
    const rc = linux.ioctl(kvm_fd, TDX_CAPABILITIES, 0);
    return rc >= 0;
}

pub fn initTDX() TDXError!void {
    if (!isTDXSupported()) {
        return TDXError.NotSupported;
    }
}

pub fn getTDXInfo() TDXInfo {
    return .{
        .trusted = isTDXSupported(),
        .attestation_supported = false,
    };
}

pub fn createTDXVMMemoryRegion(
    vm_fd: i32,
    guest_phys_addr: u64,
    size: u64,
    userspace_addr: u64,
) !void {
    const KVM_TDX_SET_MEMORY_REGION: u32 = 0x4020ae80;
    const TdxMemoryRegion = extern struct {
        slot: u32,
        flags: u32,
        guest_phys_addr: u64,
        size: u64,
        userspace_addr: u64,
    };

    const region = TdxMemoryRegion{
        .slot = 0,
        .flags = 0,
        .guest_phys_addr = guest_phys_addr,
        .size = size,
        .userspace_addr = userspace_addr,
    };

    const rc = linux.ioctl(vm_fd, KVM_TDX_SET_MEMORY_REGION, @intFromPtr(&region));
    if (rc < 0) {
        return TDXError.InitFailed;
    }
}

pub fn finalizeTDXVM(vm_fd: i32) TDXError!void {
    const KVM_TDX_FINALIZE_VM: u32 = 0x6003ae80;
    const rc = linux.ioctl(vm_fd, KVM_TDX_FINALIZE_VM, 0);
    if (rc < 0) {
        return TDXError.InitFailed;
    }
}
