const std = @import("std");

var map: std.AutoHashMap(u32, i32) = undefined;
var initialized = false;

pub fn init() void {
    map = std.AutoHashMap(u32, i32).init(std.heap.page_allocator);
    initialized = true;
}

pub fn register(cid: u32, fd: i32) void {
    if (!initialized) return;
    map.put(cid, fd) catch {};
}

pub fn get(cid: u32) ?i32 {
    if (!initialized) return null;
    return map.get(cid);
}

pub fn remove(cid: u32) void {
    if (!initialized) return;
    _ = map.remove(cid);
}
