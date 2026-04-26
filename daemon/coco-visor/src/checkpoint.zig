// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

//! Checkpoint/hibernate implementation with zstd compression.
//! Spec: Checkpoint operations save VM state to disk for later restoration.
//! Memory pages are compressed using zstd for fast compression with good ratios.

const std = @import("std");
const sc = @import("syscall.zig");

extern fn ZSTD_compressBound(srcSize: usize) usize;
extern fn ZSTD_compress(dst: [*]u8, dstCapacity: usize, src: [*]const u8, srcSize: usize, compressionLevel: c_int) usize;
extern fn ZSTD_decompress(dst: [*]u8, dstCapacity: usize, src: [*]const u8, compressedSize: usize) usize;
extern fn ZSTD_isError(code: usize) c_uint;

fn parseStringField(content: []const u8, key: []const u8) ?[]const u8 {
    const start_idx = std.mem.indexOf(u8, content, key) orelse return null;
    const value_start = start_idx + key.len;
    if (value_start >= content.len) return null;
    const end_offset = std.mem.indexOfScalar(u8, content[value_start..], '"') orelse return null;
    return content[value_start .. value_start + end_offset];
}

fn parseU64Field(content: []const u8, key: []const u8) ?u64 {
    const start_idx = std.mem.indexOf(u8, content, key) orelse return null;
    const value_start = start_idx + key.len;
    if (value_start >= content.len) return null;
    var end: usize = value_start;
    while (end < content.len and (content[end] >= '0' and content[end] <= '9')) : (end += 1) {}
    if (end == value_start) return null;
    return std.fmt.parseInt(u64, content[value_start..end], 10) catch null;
}

pub const CheckpointError = error{
    OpenFailed,
    WriteFailed,
    ReadFailed,
    CompressionFailed,
    DecompressionFailed,
    OutOfMemory,
    InvalidCheckpoint,
};

pub const CheckpointMetadata = struct {
    id: []const u8,
    memory_size: u64,
    compressed_size: u64,
    timestamp: i64,
    memory_mb: u32,
    vcpus: u32,
    kernel: []const u8,
    rootfs: []const u8,
    incremental: bool = false,
    parent_id: []const u8 = "",
};

pub const SNAPSHOT_DIR = "/var/lib/coco/checkpoints";

const CHUNK_SIZE: usize = 65536;
const COMPRESSION_LEVEL: i32 = 3;

pub const CheckpointWriter = struct {
    fd: i32,
    allocator: std.mem.Allocator,
    total_compressed_size: u64 = 0,

    pub fn init(allocator: std.mem.Allocator, fd: i32) CheckpointWriter {
        return .{
            .fd = fd,
            .allocator = allocator,
        };
    }

    pub fn writeMemory(self: *CheckpointWriter, mem_ptr: [*]u8, mem_size: u64) u64 {
        var written: u64 = 0;
        var chunk_idx: u32 = 0;

        while (written < mem_size) {
            const to_write = @min(CHUNK_SIZE, mem_size - written);
            const chunk = mem_ptr[written .. written + to_write];
            self.writeChunk(chunk, chunk_idx);
            written += to_write;
            chunk_idx += 1;
        }

        return self.total_compressed_size;
    }

    fn writeChunk(self: *CheckpointWriter, data: []const u8, chunk_idx: u32) void {
        const compressed = self.compress(data);
        self.total_compressed_size += compressed.len + 12;

        var chunk_header: [12]u8 = undefined;
        std.mem.writeInt(u32, chunk_header[0..4], chunk_idx, .little);
        std.mem.writeInt(u32, chunk_header[4..8], @intCast(data.len), .little);
        std.mem.writeInt(u32, chunk_header[8..12], @intCast(compressed.len), .little);

        _ = sc.write(self.fd, &chunk_header) catch {};
        _ = sc.write(self.fd, compressed) catch {};
    }

    fn compress(self: *CheckpointWriter, data: []const u8) []const u8 {
        const bound = ZSTD_compressBound(data.len);
        const out = self.allocator.alloc(u8, bound) catch return data;
        const result = ZSTD_compress(out.ptr, bound, data.ptr, data.len, @as(c_int, COMPRESSION_LEVEL));
        if (ZSTD_isError(result) != 0) {
            self.allocator.free(out);
            return data;
        }
        return out[0..result];
    }

    pub fn compressData(allocator: std.mem.Allocator, data: []const u8) []const u8 {
        const bound = ZSTD_compressBound(data.len);
        const out = allocator.alloc(u8, bound) catch return data;
        const result = ZSTD_compress(out.ptr, bound, data.ptr, data.len, @as(c_int, COMPRESSION_LEVEL));
        if (ZSTD_isError(result) != 0) {
            allocator.free(out);
            return data;
        }
        return out[0..result];
    }
};

pub const CheckpointReader = struct {
    fd: i32,
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator, fd: i32) CheckpointReader {
        return .{
            .fd = fd,
            .allocator = allocator,
        };
    }

    pub fn readMemory(self: *CheckpointReader, mem_ptr: [*]u8, mem_size: u64) u64 {
        var read_total: u64 = 0;
        var chunk_idx: u32 = 0;

        while (read_total < mem_size) {
            var header: [12]u8 = undefined;
            var got: usize = 0;
            while (got < 12) {
                const n = sc.read(self.fd, header[got..]) catch break;
                if (n == 0) break;
                got += n;
            }
            if (got < 12) break;

            const orig_size = std.mem.readInt(u32, header[4..8], .little);
            const comp_size = std.mem.readInt(u32, header[8..12], .little);

            const remaining = mem_size - read_total;
            const to_write = @min(@as(u64, orig_size), remaining);

            const comp_buf = self.allocator.alloc(u8, comp_size) catch return read_total;
            defer self.allocator.free(comp_buf);

            var c_got: usize = 0;
            while (c_got < comp_size) {
                const n = sc.read(self.fd, comp_buf[c_got..]) catch break;
                if (n == 0) break;
                c_got += n;
            }
            if (c_got < comp_size) break;

            const dst_slice = mem_ptr[read_total .. read_total + to_write];
            const decoded_len = ZSTD_decompress(dst_slice.ptr, dst_slice.len, comp_buf.ptr, comp_size);
            if (ZSTD_isError(decoded_len) != 0) {
                @memcpy(dst_slice[0..@min(comp_size, dst_slice.len)], comp_buf[0..@min(comp_size, dst_slice.len)]);
                read_total += @min(comp_size, dst_slice.len);
            } else {
                read_total += decoded_len;
            }

            chunk_idx += 1;
        }

        return read_total;
    }
};

pub const CheckpointManager = struct {
    base_dir: []const u8,
    allocator: std.mem.Allocator,
    dirty_pages: std.AutoHashMap(u64, bool),

    pub fn init(allocator: std.mem.Allocator) CheckpointManager {
        return .{
            .base_dir = SNAPSHOT_DIR,
            .allocator = allocator,
            .dirty_pages = std.AutoHashMap(u64, bool).init(allocator),
        };
    }

    pub fn deinit(self: *CheckpointManager) void {
        self.dirty_pages.deinit();
    }

    pub fn createCheckpoint(
        self: *CheckpointManager,
        id: []const u8,
        mem_ptr: [*]u8,
        mem_size: u64,
        metadata: CheckpointMetadata,
        vcpu_fd: i32,
    ) u64 {
        const checkpoint_dir = std.fmt.allocPrint(self.allocator, "{s}/{s}", .{ self.base_dir, id }) catch return 0;
        defer self.allocator.free(checkpoint_dir);

        sc.mkdir(checkpoint_dir, 0o755) catch {};

        const mem_file_path = std.fmt.allocPrint(self.allocator, "{s}/memory.zst", .{checkpoint_dir}) catch return 0;
        defer self.allocator.free(mem_file_path);

        const mem_fd = sc.open(mem_file_path, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644) catch return 0;
        defer sc.close(mem_fd);

        var writer = CheckpointWriter.init(self.allocator, mem_fd);

        const written = writer.writeMemory(mem_ptr, mem_size);

        const meta_path = std.fmt.allocPrint(self.allocator, "{s}/metadata.json", .{checkpoint_dir}) catch return 0;
        defer self.allocator.free(meta_path);

        var meta = metadata;
        meta.compressed_size = written;
        self.writeMetadata(meta_path, meta);

        const cpu_path = std.fmt.allocPrint(self.allocator, "{s}/cpu.bin", .{checkpoint_dir}) catch return written;
        self.saveCpuStateFor(cpu_path, vcpu_fd);

        const devices_path = std.fmt.allocPrint(self.allocator, "{s}/devices.bin", .{checkpoint_dir}) catch return written;
        self.saveDeviceState(devices_path);

        return written;
    }

    pub fn createIncrementalCheckpoint(
        self: *CheckpointManager,
        id: []const u8,
        mem_ptr: [*]u8,
        mem_size: u64,
        metadata: CheckpointMetadata,
        vcpu_fd: i32,
    ) u64 {
        var meta = metadata;
        meta.incremental = true;

        var written: u64 = 0;

        var page_idx: u64 = 0;
        while (page_idx * 4096 < mem_size) : (page_idx += 1) {
            if (self.dirty_pages.get(page_idx)) |_| {
                const page_ptr = mem_ptr + (page_idx * 4096);
                const page_data = page_ptr[0..4096];
                written += page_data.len;
            }
        }

        return self.createCheckpoint(id, mem_ptr, mem_size, meta, vcpu_fd);
    }

    pub fn restoreCheckpoint(
        self: *CheckpointManager,
        id: []const u8,
        mem_ptr: [*]u8,
    ) CheckpointMetadata {
        const checkpoint_dir = std.fmt.allocPrint(self.allocator, "{s}/{s}", .{ self.base_dir, id }) catch return .{
            .id = "",
            .memory_size = 0,
            .compressed_size = 0,
            .timestamp = 0,
            .memory_mb = 512,
            .vcpus = 2,
            .kernel = "",
            .rootfs = "",
        };
        defer self.allocator.free(checkpoint_dir);

        const meta_path = std.fmt.allocPrint(self.allocator, "{s}/metadata.json", .{checkpoint_dir}) catch return .{
            .id = "",
            .memory_size = 0,
            .compressed_size = 0,
            .timestamp = 0,
            .memory_mb = 512,
            .vcpus = 2,
            .kernel = "",
            .rootfs = "",
        };
        defer self.allocator.free(meta_path);

        const metadata = self.readMetadata(meta_path);

        const mem_file_path = std.fmt.allocPrint(self.allocator, "{s}/memory.zst", .{checkpoint_dir}) catch return metadata;
        defer self.allocator.free(mem_file_path);

        const mem_fd = sc.open(mem_file_path, .{ .ACCMODE = .RDONLY }, 0) catch return metadata;
        defer sc.close(mem_fd);

        var reader = CheckpointReader.init(self.allocator, mem_fd);
        _ = reader.readMemory(mem_ptr, metadata.memory_size);

        return metadata;
    }

    pub fn saveCpuStateFor(self: *CheckpointManager, path: []const u8, vcpu_fd: i32) void {
        _ = self;
        const linux = std.os.linux;
        const KVM_GET_REGS: u32 = 0x8090ae81;
        const KVM_GET_SREGS: u32 = 0x8138ae83;

        const fd = sc.open(path, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644) catch return;
        defer sc.close(fd);

        var regs: [144]u8 = undefined;
        _ = linux.ioctl(vcpu_fd, KVM_GET_REGS, @intFromPtr(&regs));
        var sregs: [312]u8 = undefined;
        _ = linux.ioctl(vcpu_fd, KVM_GET_SREGS, @intFromPtr(&sregs));

        var hdr: [8]u8 = undefined;
        std.mem.writeInt(u32, hdr[0..4], @sizeOf(@TypeOf(regs)), .little);
        std.mem.writeInt(u32, hdr[4..8], @sizeOf(@TypeOf(sregs)), .little);
        var w: usize = 0;
        while (w < hdr.len) w += sc.write(fd, hdr[w..]) catch return;
        w = 0;
        while (w < regs.len) w += sc.write(fd, regs[w..]) catch return;
        w = 0;
        while (w < sregs.len) w += sc.write(fd, sregs[w..]) catch return;
    }

    fn saveCpuState(self: *CheckpointManager, path: []const u8) void {
        _ = self;
        const fd = sc.open(path, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644) catch return;
        defer sc.close(fd);
        var hdr: [8]u8 = .{ 0, 0, 0, 0, 0, 0, 0, 0 };
        var w: usize = 0;
        while (w < hdr.len) w += sc.write(fd, hdr[w..]) catch return;
    }

    fn saveDeviceState(self: *CheckpointManager, path: []const u8) void {
        _ = self;
        const fd = sc.open(path, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644) catch return;
        defer sc.close(fd);
    }

    pub fn loadCpuStateInto(self: *CheckpointManager, path: []const u8, vcpu_fd: i32) void {
        _ = self;
        const linux = std.os.linux;
        const KVM_SET_REGS: u32 = 0x4090ae82;
        const KVM_SET_SREGS: u32 = 0x4138ae84;

        const fd = sc.open(path, .{ .ACCMODE = .RDONLY }, 0) catch return;
        defer sc.close(fd);

        var hdr: [8]u8 = undefined;
        var got: usize = 0;
        while (got < hdr.len) {
            const n = sc.read(fd, hdr[got..]) catch return;
            if (n == 0) return;
            got += n;
        }
        const regs_len = std.mem.readInt(u32, hdr[0..4], .little);
        const sregs_len = std.mem.readInt(u32, hdr[4..8], .little);
        if (regs_len == 0 or sregs_len == 0) return;

        var regs: [144]u8 = undefined;
        var sregs: [312]u8 = undefined;
        if (regs_len > regs.len or sregs_len > sregs.len) return;

        got = 0;
        while (got < regs_len) {
            const n = sc.read(fd, regs[got..regs_len]) catch return;
            if (n == 0) return;
            got += n;
        }
        got = 0;
        while (got < sregs_len) {
            const n = sc.read(fd, sregs[got..sregs_len]) catch return;
            if (n == 0) return;
            got += n;
        }

        _ = linux.ioctl(vcpu_fd, KVM_SET_REGS, @intFromPtr(&regs));
        _ = linux.ioctl(vcpu_fd, KVM_SET_SREGS, @intFromPtr(&sregs));
    }

    pub fn loadCpuState(self: *CheckpointManager, path: []const u8) void {
        _ = self;
        _ = path;
    }

    pub fn loadDeviceState(self: *CheckpointManager, path: []const u8) void {
        _ = self;
        _ = path;
    }

    pub fn markPageDirty(self: *CheckpointManager, page_idx: u64) void {
        self.dirty_pages.put(page_idx, true) catch {};
    }

    pub fn markPagesDirtyFromBitmap(self: *CheckpointManager, bitmap: []u64, num_pages: usize) void {
        var page_idx: usize = 0;
        while (page_idx < num_pages) : (page_idx += 1) {
            const word_idx = page_idx / 64;
            const bit_idx = page_idx % 64;
            if (word_idx < bitmap.len and (bitmap[word_idx] & (1 << bit_idx)) != 0) {
                self.markPageDirty(@intCast(page_idx));
            }
        }
    }

    pub fn clearDirtyPages(self: *CheckpointManager) void {
        self.dirty_pages.clear();
    }

    pub fn getDirtyPageCount(self: *CheckpointManager) usize {
        return self.dirty_pages.count();
    }

    fn writeMetadata(self: *CheckpointManager, path: []const u8, meta: CheckpointMetadata) void {
        const fd = sc.open(path, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644) catch return;
        defer sc.close(fd);

        var buf: [2048]u8 = undefined;
        const incremental_str: []const u8 = if (meta.incremental) "true" else "false";
        const json = std.fmt.bufPrint(&buf,
            \\{{"id":"{s}","memory_size":{d},"compressed_size":{d},"timestamp":{d},"memory_mb":{d},"vcpus":{d},"incremental":{s},"parent_id":"{s}"}}
        , .{
            meta.id,
            meta.memory_size,
            meta.compressed_size,
            meta.timestamp,
            meta.memory_mb,
            meta.vcpus,
            incremental_str,
            meta.parent_id,
        }) catch return;
        _ = self;
        var w: usize = 0;
        while (w < json.len) {
            w += sc.write(fd, json[w..]) catch return;
        }
    }

    fn readMetadata(self: *CheckpointManager, path: []const u8) CheckpointMetadata {
        const empty: CheckpointMetadata = .{
            .id = "",
            .memory_size = 0,
            .compressed_size = 0,
            .timestamp = 0,
            .memory_mb = 512,
            .vcpus = 2,
            .kernel = "",
            .rootfs = "",
        };
        const fd = sc.open(path, .{ .ACCMODE = .RDONLY }, 0) catch return empty;
        defer sc.close(fd);

        var buf: [4096]u8 = undefined;
        const n = sc.read(fd, &buf) catch return empty;
        const content = buf[0..n];

        var meta = empty;
        const id_bytes = parseStringField(content, "\"id\":\"") orelse "";
        if (id_bytes.len > 0) {
            meta.id = self.allocator.dupe(u8, id_bytes) catch "";
        }
        meta.memory_size = parseU64Field(content, "\"memory_size\":") orelse 0;
        meta.compressed_size = parseU64Field(content, "\"compressed_size\":") orelse 0;
        meta.timestamp = @intCast(parseU64Field(content, "\"timestamp\":") orelse 0);
        meta.memory_mb = @intCast(parseU64Field(content, "\"memory_mb\":") orelse 512);
        meta.vcpus = @intCast(parseU64Field(content, "\"vcpus\":") orelse 2);
        const parent = parseStringField(content, "\"parent_id\":\"") orelse "";
        if (parent.len > 0) {
            meta.parent_id = self.allocator.dupe(u8, parent) catch "";
        }
        return meta;
    }

    pub fn deleteCheckpoint(self: *CheckpointManager, id: []const u8) void {
        const checkpoint_dir = std.fmt.allocPrint(self.allocator, "{s}/{s}", .{ self.base_dir, id }) catch return;
        defer self.allocator.free(checkpoint_dir);
        sc.unlink(checkpoint_dir) catch {};
    }

    pub fn listCheckpoints(self: *CheckpointManager) []const []const u8 {
        var list: std.ArrayList([]const u8) = .empty;

        var iter = sc.openDir(self.base_dir) catch return &[_][]const u8{};
        defer iter.deinit();

        while (iter.next() catch null) |entry| {
            if (entry.kind == sc.DT_DIR) {
                const name_copy = self.allocator.dupe(u8, entry.nameSlice()) catch continue;
                list.append(self.allocator, name_copy) catch {};
            }
        }

        return list.items;
    }
};
