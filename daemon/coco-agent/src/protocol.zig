// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");

pub const MessageType = enum(u32) {
    exec = 1,
    signal = 2,
    resize = 3,
    ping = 4,
    exit = 5,
};

pub const StreamType = enum(u32) {
    stdout = 1,
    stderr = 2,
    exit = 3,
    heartbeat = 4,
};

pub const Message = struct {
    msg_type: MessageType,
    length: u32,
    payload: []u8,
};

pub const ExecMessage = struct {
    command: []u8,
    args: [][]u8,
    env: ?[][]u8 = null,
    cwd: ?[]u8 = null,
    tty: bool = false,
};

pub const SignalMessage = struct {
    pid: u32,
    signal: u32,
};

pub const ResizeMessage = struct {
    rows: u16,
    cols: u16,
};

pub const ExitMessage = struct {
    exit_code: u8,
    signal: ?u32 = null,
};

pub fn encodeExecMessage(msg: ExecMessage, buffer: []u8) !usize {
    var offset: usize = 0;

    std.mem.writeIntLittle(u32, buffer[offset..][0..4], @intFromEnum(MessageType.exec));
    offset += 4;

    const payload_len = msg.command.len + 4;
    for (msg.args) |arg| {
        // placeholder for encoding
        _ = arg;
    }

    std.mem.writeIntLittle(u32, buffer[offset..][0..4], @intCast(payload_len));
    offset += 4;

    std.mem.copyForwards(u8, buffer[offset..], msg.command);
    offset += msg.command.len;

    return offset;
}

pub fn decodeExecMessage(payload: []u8) !ExecMessage {
    var msg: ExecMessage = undefined;

    if (payload.len < 4) {
        return error.InvalidMessage;
    }

    const cmd_len = std.mem.readIntLittle(u32, payload[0..4]);
    if (payload.len < 4 + cmd_len) {
        return error.InvalidMessage;
    }

    msg.command = payload[4..][0..cmd_len];

    return msg;
}

pub fn encodeExitMessage(msg: ExitMessage, buffer: []u8) usize {
    var offset: usize = 0;

    std.mem.writeIntLittle(u32, buffer[offset..][0..4], @intFromEnum(StreamType.exit));
    offset += 4;

    buffer[offset] = msg.exit_code;
    offset += 1;

    return offset;
}

pub fn encodeStreamHeader(stream_type: StreamType, length: u32, buffer: []u8) usize {
    var offset: usize = 0;

    std.mem.writeIntLittle(u32, buffer[offset..][0..4], @intFromEnum(stream_type));
    offset += 4;

    std.mem.writeIntLittle(u32, buffer[offset..][0..4], length);
    offset += 4;

    return offset;
}
