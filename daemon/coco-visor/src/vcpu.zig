const std = @import("std");
const sc = @import("syscall.zig");
const posix = std.posix;
const kvm = @import("kvm.zig");
const memory = @import("memory.zig");

const GDT_CODE32 = 0x08;
const GDT_DATA32 = 0x10;
const GDT_ADDR = 0x500;
const IDT_ADDR = 0x520;

pub const VCpu = struct {
    fd: i32,
    kvm_run: *kvm.KvmRun,
    mmap_size: usize,
    id: u32,
    halt: std.atomic.Value(bool),

    pub fn create(vm_fd: i32, id: u32, mmap_size: usize) !VCpu {
        const vcpu_fd = try kvm.createVcpu(vm_fd, @intCast(id));
        const mmap_ptr = try posix.mmap(null, mmap_size, posix.PROT{ .READ = true, .WRITE = true }, posix.MAP{ .TYPE = .SHARED }, vcpu_fd, 0);
        return VCpu{
            .fd = vcpu_fd,
            .kvm_run = @ptrCast(mmap_ptr),
            .mmap_size = mmap_size,
            .id = id,
            .halt = .{ .raw = false },
        };
    }

    pub fn setup32(self: *VCpu, entry_point: u32, boot_params_addr: u32) !void {
        var sregs = try kvm.getSregs(self.fd);

        const seg_cs: kvm.KvmSegment = .{
            .base = 0,
            .limit = 0xFFFFFFFF,
            .selector = GDT_CODE32,
            .type = 0xA,
            .present = 1,
            .dpl = 0,
            .db = 1,
            .s = 1,
            .l = 0,
            .g = 1,
            .avl = 0,
            .unusable = 0,
            .padding = 0,
        };

        const seg_ds: kvm.KvmSegment = .{
            .base = 0,
            .limit = 0xFFFFFFFF,
            .selector = GDT_DATA32,
            .type = 0x3,
            .present = 1,
            .dpl = 0,
            .db = 1,
            .s = 1,
            .l = 0,
            .g = 1,
            .avl = 0,
            .unusable = 0,
            .padding = 0,
        };

        sregs.cs = seg_cs;
        sregs.ds = seg_ds;
        sregs.es = seg_ds;
        sregs.fs = seg_ds;
        sregs.gs = seg_ds;
        sregs.ss = seg_ds;

        sregs.gdt.base = GDT_ADDR;
        sregs.gdt.limit = 23;
        sregs.idt.base = IDT_ADDR;
        sregs.idt.limit = 0;

        sregs.cr0 = 0x1;
        sregs.cr2 = 0;
        sregs.cr3 = 0;
        sregs.cr4 = 0;

        try kvm.setSregs(self.fd, &sregs);

        var regs = try kvm.getRegs(self.fd);
        regs.rip = entry_point;
        regs.rsi = boot_params_addr;
        regs.rsp = 0x8000;
        regs.rbp = 0;
        regs.rflags = 0x2;

        try kvm.setRegs(self.fd, &regs);
    }

    pub fn runLoop(self: *VCpu) !void {
        while (!self.halt.load(.seq_cst)) {
            try kvm.runVcpu(self.fd);

            const exit_reason = self.kvm_run.exit_reason;
            switch (exit_reason) {
                kvm.KVM_EXIT_IO => self.handleIo(),
                kvm.KVM_EXIT_HLT => {
                    self.halt.store(true, .seq_cst);
                    return;
                },
                kvm.KVM_EXIT_SHUTDOWN => return,
                kvm.KVM_EXIT_INTERNAL_ERROR => return error.KVMInternalError,
                else => {
                    std.debug.print("Unknown KVM exit reason: {d}\n", .{exit_reason});
                },
            }
        }
    }

    fn handleIo(self: *VCpu) void {
        const io_data = @as(*const struct {
            direction: u8,
            size: u8,
            port: u16,
            count: u32,
            data_offset: u64,
        }, @ptrCast(&self.kvm_run.exit_data[0]));

        if (io_data.port == 0x3F8) {
            if (io_data.direction == 1) {
                const data_ptr = @as([*]u8, @ptrCast(self.kvm_run)) + io_data.data_offset;
                const data = data_ptr[0..io_data.size];
                _ = sc.write(2, data) catch {};
            }
        }
    }

    pub fn deinit(self: *VCpu) void {
        const aligned_ptr = @as([*]align(std.heap.page_size_min) u8, @ptrCast(@alignCast(self.kvm_run)));
        const slice = aligned_ptr[0..self.mmap_size];
        posix.munmap(slice);
        sc.close(self.fd);
    }
};
