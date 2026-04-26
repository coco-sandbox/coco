const std = @import("std");
const linux = std.os.linux;

const CAP_CHOWN: u32 = 0;
const CAP_DAC_OVERRIDE: u32 = 1;
const CAP_DAC_READ_SEARCH: u32 = 2;
const CAP_FOWNER: u32 = 3;
const CAP_FSETID: u32 = 4;
const CAP_KILL: u32 = 5;
const CAP_SETGID: u32 = 6;
const CAP_SETUID: u32 = 7;
const CAP_SETPCAP: u32 = 8;
const CAP_LINUX_IMMUTABLE: u32 = 9;
const CAP_NET_BIND_SERVICE: u32 = 10;
const CAP_NET_BROADCAST: u32 = 11;
const CAP_NET_ADMIN: u32 = 12;
const CAP_NET_RAW: u32 = 13;
const CAP_IPC_LOCK: u32 = 14;
const CAP_IPC_OWNER: u32 = 15;
const CAP_SYS_MODULE: u32 = 16;
const CAP_SYS_RAWIO: u32 = 17;
const CAP_SYS_CHROOT: u32 = 18;
const CAP_SYS_PTRACE: u32 = 19;
const CAP_SYS_PACCT: u32 = 20;
const CAP_SYS_ADMIN: u32 = 21;
const CAP_SYS_BOOT: u32 = 22;
const CAP_SYS_NICE: u32 = 23;
const CAP_SYS_RESOURCE: u32 = 24;
const CAP_SYS_TIME: u32 = 25;
const CAP_SYS_TTY_CONFIG: u32 = 26;
const CAP_MKNOD: u32 = 27;
const CAP_LEASE: u32 = 28;
const CAP_AUDIT_WRITE: u32 = 29;
const CAP_AUDIT_CONTROL: u32 = 30;
const CAP_SETFCAP: u32 = 31;

const caps_drop_all = [_]u32{
    CAP_SYS_ADMIN,
    CAP_NET_ADMIN,
    CAP_SYS_MODULE,
    CAP_SYS_RAWIO,
    CAP_SYS_PTRACE,
    CAP_SYS_TIME,
    CAP_SYS_BOOT,
    CAP_AUDIT_CONTROL,
    CAP_SYS_PACCT,
    CAP_SYS_NICE,
    CAP_SYS_RESOURCE,
    CAP_SYS_TTY_CONFIG,
    CAP_MKNOD,
    CAP_LEASE,
    CAP_SETFCAP,
};

pub const CapabilityManager = struct {
    pub fn dropAllCapabilities() void {
        const current = getCurrentCaps() catch return;

        var effective = current.effective;
        var permitted = current.permitted;
        var inheritable = current.inheritable;

        for (caps_drop_all) |cap| {
            effective &= ~(1 << cap);
            permitted &= ~(1 << cap);
            inheritable &= ~(1 << cap);
        }

        setCaps(.{
            .effective = effective,
            .permitted = permitted,
            .inheritable = inheritable,
        }) catch return;
    }

    pub fn dropCapability(cap: u32) void {
        var current = getCurrentCaps() catch return;

        current.effective &= ~(1 << cap);
        current.permitted &= ~(1 << cap);

        setCaps(current) catch return;
    }

    const CapSet = struct {
        effective: u32,
        permitted: u32,
        inheritable: u32,
    };

    fn getCurrentCaps() !CapSet {
        var header: linux.__user_cap_header_struct = undefined;
        var data: linux.__user_cap_data_struct = undefined;

        header.version = 0x19980330;
        header.pid = 0;

        const ret = linux.capget(&header, &data);
        if (ret != 0) {
            return error.CapgetFailed;
        }

        return CapSet{
            .effective = data.effective,
            .permitted = data.permitted,
            .inheritable = data.inheritable,
        };
    }

    fn setCaps(caps: CapSet) !void {
        var header: linux.__user_cap_header_struct = undefined;
        var data: linux.__user_cap_data_struct = undefined;

        header.version = 0x19980330;
        header.pid = 0;

        data.effective = caps.effective;
        data.permitted = caps.permitted;
        data.inheritable = caps.inheritable;

        const ret = linux.capset(&header, &data);
        if (ret != 0) {
            return error.CapsetFailed;
        }
    }

    pub fn hasCapability(cap: u32) bool {
        const current = getCurrentCaps() catch return false;
        return (current.effective & (1 << cap)) != 0;
    }
};
