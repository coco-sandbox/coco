const std = @import("std");
const posix = std.posix;
const linux = std.os.linux;

pub const KVM_GET_API_VERSION = 0x0000AE00;
pub const KVM_CREATE_VM = 0x0000AE01;
pub const KVM_CREATE_VCPU = 0x0000AE41;
pub const KVM_GET_VCPU_MMAP_SIZE = 0x0000AE04;
pub const KVM_SET_USER_MEMORY_REGION = 0x4020AE46;
pub const KVM_RUN = 0x0000AE80;
pub const KVM_GET_REGS = 0x8090AE81;
pub const KVM_SET_REGS = 0x4090AE82;
pub const KVM_GET_SREGS = 0x8138AE83;
pub const KVM_SET_SREGS = 0x4138AE84;

pub const KVM_EXIT_IO = 2;
pub const KVM_EXIT_HLT = 5;
pub const KVM_EXIT_MMIO = 6;
pub const KVM_EXIT_SHUTDOWN = 8;
pub const KVM_EXIT_INTERNAL_ERROR = 17;

pub const KvmUserspaceMemoryRegion = extern struct {
    slot: u32,
    flags: u32,
    guest_phys_addr: u64,
    memory_size: u64,
    userspace_addr: u64,
};

pub const KvmSegment = extern struct {
    base: u64,
    limit: u32,
    selector: u16,
    type: u8,
    present: u8,
    dpl: u8,
    db: u8,
    s: u8,
    l: u8,
    g: u8,
    avl: u8,
    unusable: u8,
    padding: u8,
};

pub const KvmDtable = extern struct {
    base: u64,
    limit: u16,
    padding: [3]u16,
};

pub const KvmRegs = extern struct {
    rax: u64,
    rbx: u64,
    rcx: u64,
    rdx: u64,
    rsi: u64,
    rdi: u64,
    rsp: u64,
    rbp: u64,
    r8: u64,
    r9: u64,
    r10: u64,
    r11: u64,
    r12: u64,
    r13: u64,
    r14: u64,
    r15: u64,
    rip: u64,
    rflags: u64,
};

pub const KvmSregs = extern struct {
    cs: KvmSegment,
    ds: KvmSegment,
    es: KvmSegment,
    fs: KvmSegment,
    gs: KvmSegment,
    ss: KvmSegment,
    tr: KvmSegment,
    ldt: KvmSegment,
    gdt: KvmDtable,
    idt: KvmDtable,
    cr0: u64,
    cr2: u64,
    cr3: u64,
    cr4: u64,
    cr8: u64,
    efer: u64,
    apic_base: u64,
    interrupt_bitmap: [4]u64,
};

pub const KvmRun = extern struct {
    request_interrupt_window: u8,
    immediate_exit: u8,
    padding1: [6]u8,
    exit_reason: u32,
    ready_for_interrupt_injection: u8,
    if_flag: u8,
    flags: u16,
    cr8: u64,
    apic_base: u64,
    exit_data: [256]u8,
};

fn ioctl(fd: i32, request: u32, arg: usize) !i32 {
    const rc = linux.ioctl(fd, request, arg);
    if (rc < 0) return error.IoctlFailed;
    return @intCast(rc);
}

pub fn open() !i32 {
    return try std.posix.open("/dev/kvm", .{ .ACCMODE = .RDWR }, 0);
}

pub fn createVm(kvm_fd: i32) !i32 {
    return try ioctl(kvm_fd, KVM_CREATE_VM, 0);
}

pub fn createVcpu(vm_fd: i32, vcpu_id: i32) !i32 {
    return try ioctl(vm_fd, KVM_CREATE_VCPU, @intCast(vcpu_id));
}

pub fn getVcpuMmapSize(kvm_fd: i32) !usize {
    const rc = try ioctl(kvm_fd, KVM_GET_VCPU_MMAP_SIZE, 0);
    return @intCast(rc);
}

pub fn setUserMemoryRegion(vm_fd: i32, region: *const KvmUserspaceMemoryRegion) !void {
    _ = try ioctl(vm_fd, KVM_SET_USER_MEMORY_REGION, @intFromPtr(region));
}

pub fn runVcpu(vcpu_fd: i32) !void {
    _ = try ioctl(vcpu_fd, KVM_RUN, 0);
}

pub fn getRegs(vcpu_fd: i32) !KvmRegs {
    var regs: KvmRegs = undefined;
    _ = try ioctl(vcpu_fd, KVM_GET_REGS, @intFromPtr(&regs));
    return regs;
}

pub fn setRegs(vcpu_fd: i32, regs: *const KvmRegs) !void {
    _ = try ioctl(vcpu_fd, KVM_SET_REGS, @intFromPtr(regs));
}

pub fn getSregs(vcpu_fd: i32) !KvmSregs {
    var sregs: KvmSregs = undefined;
    _ = try ioctl(vcpu_fd, KVM_GET_SREGS, @intFromPtr(&sregs));
    return sregs;
}

pub fn setSregs(vcpu_fd: i32, sregs: *const KvmSregs) !void {
    _ = try ioctl(vcpu_fd, KVM_SET_SREGS, @intFromPtr(sregs));
}
