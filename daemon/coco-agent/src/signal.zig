// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

const std = @import("std");
const posix = std.posix;

pub const SignalHandler = fn (sig: c_int) void;

pub const SignalSet = posix.sigset_t;

pub fn initSignalSet() SignalSet {
    var set: SignalSet = undefined;
    posix.sigemptyset(&set);
    return set;
}

pub fn addSignal(set: *SignalSet, sig: c_int) void {
    posix.sigaddset(set, sig);
}

pub fn blockSignal(sig: c_int) void {
    var set = initSignalSet();
    addSignal(&set, sig);
    posix.sigprocmask(.BLOCK, &set, null);
}

pub fn unblockSignal(sig: c_int) void {
    var set = initSignalSet();
    addSignal(&set, sig);
    posix.sigprocmask(.UNBLOCK, &set, null);
}

pub fn handleSignal(sig: c_int, handler: SignalHandler) void {
    var act: posix.Sigaction = .{
        .handler = .{ .handler = handler },
        .mask = initSignalSet(),
        .flags = 0,
    };
    posix.sigaction(sig, &act, null);
}

pub fn sendSignal(pid: posix.pid_t, sig: c_int) void {
    posix.kill(pid, sig) catch {};
}

pub const Signal = enum(c_int) {
    sighup = posix.SIG.HUP,
    sigint = posix.SIG.INT,
    sigquit = posix.SIG.QUIT,
    sigill = posix.SIG.ILL,
    sigtrap = posix.SIG.TRAP,
    sigabrt = posix.SIG.ABRT,
    sigbus = posix.SIG.BUS,
    sigfpe = posix.SIG.FPE,
    sigkill = posix.SIG.KILL,
    sigusr1 = posix.SIG.USR1,
    sigsegv = posix.SIG.SEGV,
    sigusr2 = posix.SIG.USR2,
    sigpipe = posix.SIG.PIPE,
    sigalrm = posix.SIG.ALRM,
    sigterm = posix.SIG.TERM,
    sigstkflt = posix.SIG.STKFLT,
    sigchld = posix.SIG.CHLD,
    sigcont = posix.SIG.CONT,
    sigstop = posix.SIG.STOP,
    sigtstp = posix.SIG.TSTP,
    sigttin = posix.SIG.TTIN,
    sigttou = posix.SIG.TTOU,
    sigurg = posix.SIG.URG,
    sigxcpu = posix.SIG.XCPU,
    sigxfsz = posix.SIG.XFSZ,
    sigvtalrm = posix.SIG.VTALRM,
    sigprof = posix.SIG.PROF,
    sigwinch = posix.SIG.WINCH,
    sigio = posix.SIG.IO,
    sigpwr = posix.SIG.PWR,
    sigsys = posix.SIG.SYS,
};

pub fn signalName(sig: c_int) [:0]const u8 {
    return switch (sig) {
        posix.SIG.HUP => "SIGHUP",
        posix.SIG.INT => "SIGINT",
        posix.SIG.QUIT => "SIGQUIT",
        posix.SIG.ILL => "SIGILL",
        posix.SIG.TRAP => "SIGTRAP",
        posix.SIG.ABRT => "SIGABRT",
        posix.SIG.BUS => "SIGBUS",
        posix.SIG.FPE => "SIGFPE",
        posix.SIG.KILL => "SIGKILL",
        posix.SIG.USR1 => "SIGUSR1",
        posix.SIG.SEGV => "SIGSEGV",
        posix.SIG.USR2 => "SIGUSR2",
        posix.SIG.PIPE => "SIGPIPE",
        posix.SIG.ALRM => "SIGALRM",
        posix.SIG.TERM => "SIGTERM",
        else => "UNKNOWN",
    };
}
