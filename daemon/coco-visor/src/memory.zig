const std = @import("std");
const posix = std.posix;

const GDT_ADDR: u64 = 0x500;
const IDT_ADDR: u64 = 0x520;
const BOOT_PARAMS_ADDR: u64 = 0x7000;
const CMDLINE_ADDR: u64 = 0x20000;
pub const KERNEL_LOAD_ADDR: u64 = 0x100000;
const INITRD_LOAD_ADDR: u64 = 0x4000000;
const LOWMEM_END: u64 = 0x9FC00;

const E820_RAM: u32 = 1;
const E820_RESERVED: u32 = 2;

const E820Entry = extern struct {
    addr: u64,
    size: u64,
    type: u32,
};

const BootParams = extern struct {
    padding1: [0x1F1]u8,
    setup_sects: u8,
    root_flags: u16,
    syssize: u32,
    ram_size: u16,
    video_mode: u16,
    root_dev: u16,
    boot_flag: u16,
    jump: u16,
    header: u32,
    version: u16,
    realmode_swtch: u32,
    start_sys: u16,
    kernel_version: u16,
    type_of_loader: u8,
    loadflags: u8,
    setup_move_size: u16,
    code32_start: u32,
    ramdisk_image: u32,
    ramdisk_size: u32,
    bootsect_kludge: u32,
    heap_end_ptr: u16,
    ext_loader_ver: u8,
    ext_loader_type: u8,
    cmd_line_ptr: u32,
    initrd_addr_max: u32,
    kernel_alignment: u32,
    relocatable_kernel: u8,
    min_alignment: u8,
    xload_flags: u16,
    cmdline_size: u32,
    hardware_subvend_id: u32,
    hardware_subprod_id: u32,
    hardware_bios_vendor: u32,
    hardware_bios_date: u32,
    payload_offset: u32,
    payload_length: u32,
    setup_data: u64,
    pref_address: u64,
    init_size: u32,
    handover_offset: u32,
    kernel_info_offset: u32,
    padding2: [0x290 - 0x23E]u8,
    e820_entries: u32,
    e820_table: [128]E820Entry,
};

pub const GuestMemory = struct {
    ptr: [*]u8,
    size: u64,

    pub fn init(size_mb: u32) !GuestMemory {
        const size = @as(u64, size_mb) * 1024 * 1024;
        const ptr = try posix.mmap(
            null,
            size,
            posix.PROT.READ | posix.PROT.WRITE,
            posix.MAP{ .TYPE = .SHARED, .ANONYMOUS = true },
            -1,
            0,
        );
        return GuestMemory{
            .ptr = @ptrCast(ptr),
            .size = size,
        };
    }

    pub fn deinit(self: *GuestMemory) void {
        const aligned_ptr = @as([*]align(std.heap.page_size_min) u8, @alignCast(self.ptr));
        const slice = aligned_ptr[0..self.size];
        posix.munmap(slice);
    }

    pub fn hostAddr(self: *const GuestMemory, gpa: u64) [*]u8 {
        return self.ptr + gpa;
    }

    pub fn sliceAt(self: *const GuestMemory, gpa: u64, len: usize) []u8 {
        return (self.ptr + gpa)[0..len];
    }
};

pub fn loadKernel(mem: *const GuestMemory, kernel_path: []const u8) !u64 {
    const file = try std.fs.openFileAbsolute(kernel_path, .{});
    defer file.close();

    const stat = try file.stat();
    if (stat.size < 512)
        return error.KernelTooSmall;

    var boot_header: [64]u8 = undefined;
    try file.seekTo(0x1F1);
    const n = try file.readAll(&boot_header);
    if (n < 16)
        return error.InvalidKernel;

    const header = std.mem.readInt(u32, boot_header[0x11..0x15], .little);
    if (header != 0x53726448)
        return error.InvalidKernelHeader;

    const setup_sects = boot_header[0];
    const setup_size = (@as(u64, setup_sects) + 1) * 512;

    if (setup_size >= stat.size)
        return error.InvalidSetupSize;

    try file.seekTo(setup_size);

    const kernel_size = stat.size - setup_size;
    const kernel_dst = mem.sliceAt(KERNEL_LOAD_ADDR, @intCast(kernel_size));
    const bytes_read = try file.readAll(kernel_dst);

    if (bytes_read != kernel_size)
        return error.KernelReadFailed;

    return KERNEL_LOAD_ADDR;
}

pub fn loadInitrd(mem: *const GuestMemory, initrd_path: []const u8) !struct { addr: u64, size: u64 } {
    if (initrd_path.len == 0)
        return .{ .addr = 0, .size = 0 };

    const file = try std.fs.openFileAbsolute(initrd_path, .{});
    defer file.close();

    const stat = try file.stat();
    if (stat.size == 0)
        return .{ .addr = 0, .size = 0 };

    const initrd_dst = mem.sliceAt(INITRD_LOAD_ADDR, @intCast(stat.size));
    const bytes_read = try file.readAll(initrd_dst);

    if (bytes_read != stat.size)
        return error.InitrdReadFailed;

    return .{ .addr = INITRD_LOAD_ADDR, .size = stat.size };
}

pub fn setupBootParams(mem: *const GuestMemory, cmdline: []const u8, mem_size_mb: u32, initrd_addr: u64, initrd_size: u64) !void {
    const bp_slice = mem.sliceAt(BOOT_PARAMS_ADDR, @sizeOf(BootParams));
    @memset(bp_slice, 0);

    var bp = @as(*BootParams, @ptrCast(@alignCast(bp_slice.ptr)));

    bp.type_of_loader = 0xFF;
    bp.loadflags |= 0x01;
    bp.loadflags |= 0x80;
    bp.heap_end_ptr = 0xFE00;

    if (cmdline.len > 0) {
        const cmd_slice = mem.sliceAt(CMDLINE_ADDR, cmdline.len + 1);
        @memcpy(cmd_slice[0..cmdline.len], cmdline);
        cmd_slice[cmdline.len] = 0;
        bp.cmd_line_ptr = @intCast(CMDLINE_ADDR);
    }

    bp.ramdisk_image = @intCast(initrd_addr);
    bp.ramdisk_size = @intCast(initrd_size);
    bp.video_mode = 0xFFFF;

    bp.e820_table[0] = .{
        .addr = 0x0,
        .size = LOWMEM_END,
        .type = E820_RAM,
    };

    bp.e820_table[1] = .{
        .addr = LOWMEM_END,
        .size = 0xA0000 - LOWMEM_END,
        .type = E820_RESERVED,
    };

    bp.e820_table[2] = .{
        .addr = 0xF0000,
        .size = 0x10000,
        .type = E820_RESERVED,
    };

    const mem_bytes = @as(u64, mem_size_mb) * 1024 * 1024;
    bp.e820_table[3] = .{
        .addr = 0x100000,
        .size = mem_bytes - 0x100000,
        .type = E820_RAM,
    };

    bp.e820_entries = 4;
}

pub fn setupGdt(mem: *const GuestMemory) void {
    const gdt_slice = mem.sliceAt(GDT_ADDR, 24);

    @memset(gdt_slice, 0);

    const gdt: [3][8]u8 = .{
        .{ 0, 0, 0, 0, 0, 0, 0, 0 },
        .{ 0xFF, 0xFF, 0, 0, 0, 0x9A, 0xCF, 0 },
        .{ 0xFF, 0xFF, 0, 0, 0, 0x92, 0xCF, 0 },
    };

    for (0..3) |i| {
        @memcpy(gdt_slice[i * 8 .. (i + 1) * 8], &gdt[i]);
    }
}
